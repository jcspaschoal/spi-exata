package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/types/role"
	"github.com/jcpaschoal/spi-exata/foundation/logger"
)

// Erros padronizados do pacote de autenticação
var (
	ErrForbidden    = errors.New("attempted action is not allowed")
	ErrKIDMissing   = errors.New("kid missing from token header")
	ErrKIDMalformed = errors.New("kid in token header is malformed")
	ErrUserDisabled = errors.New("user is disabled")
	ErrInvalidRole  = errors.New("token contains an invalid role")
)

// Claims represents the authorization claims transmitted via a JWT.
type Claims struct {
	jwt.RegisteredClaims
	TenantID    string `json:"tenant_id,omitempty"`
	DashboardID string `json:"dashboard_id"`
	Role        string `json:"role"`
}

// KeyLookup declares a method set of behavior for looking up
// private and public keys for JWT use.
type KeyLookup interface {
	PrivateKey(kid string) (key string, err error)
	PublicKey(kid string) (key string, err error)
}

// Config represents information required to initialize auth.
type Config struct {
	Log       *logger.Logger
	UserBus   *userbus.Core // Usado para validar se o usuário está ativo/enabled
	KeyLookup KeyLookup
	Issuer    string
}

// Auth is used to authenticate clients.
type Auth struct {
	log       *logger.Logger
	keyLookup KeyLookup
	userBus   *userbus.Core
	method    jwt.SigningMethod
	parser    *jwt.Parser
	issuer    string
}

// New creates an Auth to support authentication/authorization.
func New(cfg Config) *Auth {
	return &Auth{
		log:       cfg.Log,
		keyLookup: cfg.KeyLookup,
		userBus:   cfg.UserBus,
		method:    jwt.GetSigningMethod(jwt.SigningMethodRS256.Name),
		parser:    jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})),
		issuer:    cfg.Issuer,
	}
}

// Issuer provides the configured issuer used to authenticate tokens.
func (a *Auth) Issuer() string {
	return a.issuer
}

// GenerateToken generates a signed JWT token string representing the user Claims.
// Aceita role.Role tipada para garantir integridade.
func (a *Auth) GenerateToken(kid string, tenantID uuid.UUID, userID uuid.UUID, dashboardID uuid.UUID, r role.Role) (string, error) {

	var tid string
	if tenantID != uuid.Nil {
		tid = tenantID.String()
	}

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    a.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		TenantID:    tid,
		DashboardID: dashboardID.String(),
		Role:        r.String(),
	}

	token := jwt.NewWithClaims(a.method, claims)
	token.Header["kid"] = kid

	privateKeyPEM, err := a.keyLookup.PrivateKey(kid)
	if err != nil {
		return "", fmt.Errorf("private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return "", fmt.Errorf("parsing private key from PEM: %w", err)
	}

	str, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return str, nil
}

// Authenticate processes the token to validate the sender's token is valid.
func (a *Auth) Authenticate(ctx context.Context, bearerToken string) (Claims, error) {
	if !strings.HasPrefix(bearerToken, "Bearer ") {
		return Claims{}, errors.New("expected authorization header format: Bearer <token>")
	}

	jwtUnverified := bearerToken[7:]

	var claims Claims
	token, _, err := a.parser.ParseUnverified(jwtUnverified, &claims)
	if err != nil {
		return Claims{}, fmt.Errorf("error parsing token: %w", err)
	}

	kidRaw, exists := token.Header["kid"]
	if !exists {
		return Claims{}, ErrKIDMissing
	}

	kid, ok := kidRaw.(string)
	if !ok {
		return Claims{}, ErrKIDMalformed
	}

	pem, err := a.keyLookup.PublicKey(kid)
	if err != nil {
		return Claims{}, fmt.Errorf("fetching public key for kid %q: %w", kid, err)
	}

	if err := a.verifySignatureAndClaims(jwtUnverified, pem); err != nil {
		a.log.Info(ctx, "**Authenticate-FAILED**", "token", jwtUnverified, "userID", claims.Subject)
		return Claims{}, fmt.Errorf("authentication failed: %w", err)
	}

	// Valida se a Role que está no token é uma Role conhecida pelo sistema.
	if _, err := role.Parse(claims.Role); err != nil {
		return Claims{}, ErrInvalidRole
	}

	// Verifica no banco se o usuário ainda está ativo/habilitado
	if err := a.isUserEnabled(ctx, claims); err != nil {
		return Claims{}, fmt.Errorf("user not enabled: %w", err)
	}

	return claims, nil
}

// Authorize checks if the claims possess ONE OF the required roles.
// This allows a route to be accessible by multiple roles (e.g., Admin OR Analyst).
func (a *Auth) Authorize(ctx context.Context, claims Claims, allowedRoles ...role.Role) error {
	// Se nenhuma role for passada na rota, bloqueia por padrão (Secure by Default).
	if len(allowedRoles) == 0 {
		return fmt.Errorf("%w: no roles authorized for this endpoint", ErrForbidden)
	}

	for _, r := range allowedRoles {
		if claims.Role == r.String() {
			return nil // Match found! Access granted.
		}
	}

	return fmt.Errorf("%w: user role %q is not in the allowed list %v", ErrForbidden, claims.Role, allowedRoles)
}

// isUserEnabled checks if the user is active in the database.
func (a *Auth) isUserEnabled(ctx context.Context, claims Claims) error {
	if a.userBus == nil {
		return nil
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return fmt.Errorf("parsing user ID %q from claims: %w", claims.Subject, err)
	}

	usr, err := a.userBus.QueryByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("query user: %w", err)
	}

	if !usr.Enabled {
		return ErrUserDisabled
	}

	return nil
}

// verifySignatureAndClaims parses the token with the public key, validates the signature, and checks the issuer claim.
func (a *Auth) verifySignatureAndClaims(tokenStr, pemStr string) error {
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pemStr))
	if err != nil {
		return fmt.Errorf("parsing public key: %w", err)
	}

	var claims Claims
	token, err := a.parser.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return fmt.Errorf("validating token signature: %w", err)
	}

	if !token.Valid {
		return errors.New("token is invalid")
	}

	if claims.Issuer != a.issuer {
		return fmt.Errorf("invalid issuer: expected %q, got %q", a.issuer, claims.Issuer)
	}

	return nil
}

// verifySignatureAndClaims parses the token with the public key, validates the signature, and checks the issuer claim.
func (a *Auth) Login(ctx context.Context, email mail.Address, password string) (userbus.User, error) {

	usr, err := a.userBus.Authenticate(ctx, email, password)

	if err != nil {
		return userbus.User{}, fmt.Errorf("invalid credentials: %w", err)
	}

	return usr, nil
}

func ExtractDomain(host string) string {
	if host, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return host
}
