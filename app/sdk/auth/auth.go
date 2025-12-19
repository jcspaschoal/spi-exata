package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/types/actions"
	"github.com/jcpaschoal/spi-exata/business/types/resource"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
)

var (
	ErrForbidden             = errors.New("attempted action is not allowed")
	ErrKIDMissing            = errors.New("kid missing from token header")
	ErrKIDMalformed          = errors.New("kid in token header is malformed")
	ErrUserDisabled          = errors.New("user is disabled")
	ErrInvalidAuthentication = errors.New("policy evaluation failed for authentication")
	ErrInvalidAuthorization  = errors.New("policy evaluation failed for authorization")
	ErrInvalidID             = errors.New("ID is not in its proper form")
)

// Claims represents the authorization claims transmitted via a JWT.
type Claims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// KeyLookup declares a method set of behavior for looking up
// private and public keys for JWT use. The return could be a
// PEM encoded string or a JWS based key.
type KeyLookup interface {
	PrivateKey(kid string) (key string, err error)
	PublicKey(kid string) (key string, err error)
}

// Config represents information required to initialize auth.
type Config struct {
	Log       *logger.Logger
	UserBus   userbus.Core
	KeyLookup KeyLookup
	Issuer    string
}

// Auth is used to authenticate clients. It can generate a token for a
// set of user claims and recreate the claims by parsing the token.
type Auth struct {
	log       *logger.Logger
	keyLookup KeyLookup
	userBus   *userbus.Core
	enforcer  *casbin.Enforcer
	method    jwt.SigningMethod
	parser    *jwt.Parser
	issuer    string
}

// New creates an Auth to support authentication/authorization.
func New(cfg Config) *Auth {
	return &Auth{
		log:       cfg.Log,
		keyLookup: cfg.KeyLookup,
		userBus:   &cfg.UserBus,
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
func (a *Auth) GenerateToken(kid string, claims Claims) (string, error) {
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

	if err := a.isUserEnabled(ctx, claims); err != nil {
		return Claims{}, fmt.Errorf("user not enabled: %w", err)
	}

	return claims, nil
}

// Authorize performs a two-stage access verification (Double Guard):
// 1. Resource Check: Validates if the user has the required permission for the resource type (e.g., DASHBOARD).
// 2. Instance Check: If a resourceID is provided, validates if the user has permission for that specific instance (ACL).
// Returns nil if authorized, otherwise returns a wrapped ErrInvalidAuthorization.
func (a *Auth) Authorize(userID uuid.UUID, res resource.Resource, act actions.Action, resourceID string) error {
	uid := userID.String()
	action := act.String()

	// Stage 1: Functional access check using resource.
	if err := a.performCheck(uid, res.String(), action, "resource"); err != nil {
		return err
	}

	// Stage 2: Instance-level access check using Resource UUID.
	if resourceID != "" {

		resourceID, err := uuid.Parse(resourceID)
		if err != nil {
			return errs.New(errs.FailedPrecondition, fmt.Errorf("ID is not in its proper form : %w", err))
		}

		if err := a.performCheck(uid, resourceID.String(), action, "instance"); err != nil {
			return err
		}
	}

	return nil
}

// performCheck is a private helper that wraps the Casbin Enforce logic.
// It centralizes error handling and boolean evaluation to keep the primary flow lean.
func (a *Auth) performCheck(sub, obj, act, scope string) error {
	allowed, err := a.enforcer.Enforce(sub, obj, act)
	if err != nil {
		return fmt.Errorf("authz internal error during %s check: %w", scope, err)
	}

	if !allowed {
		return fmt.Errorf("%w: user %s lacks %s permission on %s (%s scope)",
			ErrInvalidAuthorization, sub, act, obj, scope)
	}

	return nil
}

// isUserEnabled hits the database and checks the user is not disabled. If the
// no database connection was provided, this check is skipped.
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

func (a *Auth) AddPolicy(ctx context.Context, userID uuid.UUID, res resource.Resource, act actions.Action, resourceID *uuid.UUID) error {
	obj := a.resolveObject(res, resourceID)
	sub := userID.String()
	action := act.String()

	// AddPolicy returns false if the rule already exists.
	// We usually don't treat this as an error, but you can if strictness is required.
	added, err := a.enforcer.AddPolicy(sub, obj, action)
	if err != nil {
		return fmt.Errorf("failed to add policy to enforcer: %w", err)
	}

	if !added {
		a.log.Info(ctx, "policy already exists", "sub", sub, "obj", obj, "act", action)
	}

	return nil
}

// RemovePolicy removes an existing permission rule from the enforcer.
func (a *Auth) RemovePolicy(ctx context.Context, userID uuid.UUID, res resource.Resource, act actions.Action, resourceID *uuid.UUID) error {
	obj := a.resolveObject(res, resourceID)
	sub := userID.String()
	action := act.String()

	removed, err := a.enforcer.RemovePolicy(sub, obj, action)
	if err != nil {
		return fmt.Errorf("failed to remove policy from enforcer: %w", err)
	}

	if !removed {
		return fmt.Errorf("policy not found to remove: %s, %s, %s", sub, obj, action)
	}

	return nil
}

// UpdatePolicy updates a permission by modifying the action for a specific user and resource.
// Since Casbin doesn't have a direct "Update specific field" for SQL adapters efficiently,
// we implement this as an atomic Remove + Add.
func (a *Auth) UpdatePolicy(ctx context.Context, userID uuid.UUID, res resource.Resource, resourceID *uuid.UUID, oldAct, newAct actions.Action) error {
	if err := a.RemovePolicy(ctx, userID, res, oldAct, resourceID); err != nil {
		return fmt.Errorf("update failed (remove step): %w", err)
	}

	// 2. Add the new rule
	if err := a.AddPolicy(ctx, userID, res, newAct, resourceID); err != nil {
		return fmt.Errorf("update failed (add step): %w", err)
	}

	return nil
}

// resolveObject determines whether the policy target is a generic Resource Tag
// or a specific Instance UUID based on the input.
func (a *Auth) resolveObject(res resource.Resource, resourceID *uuid.UUID) string {
	if resourceID != nil {
		return resourceID.String() // Instance Scope (e.g., "550e8400-...")
	}
	return res.String()
}
