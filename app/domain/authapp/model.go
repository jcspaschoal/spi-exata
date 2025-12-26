package authapp

import (
	"encoding/json"
	"fmt"

	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
)

type Token struct {
	Token string `json:"token"`
}

// Encode implements the web.Encoder interface.
func (t Token) Encode() ([]byte, string, error) {
	data, err := json.Marshal(t)
	return data, "application/json", err
}

func toAppToken(token string) Token {
	return Token{
		Token: token,
	}
}

type Login struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Decode implements the web.Decoder interface.
func (app *Login) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

// Validate checks the data in the model is considered clean.
func (app Login) Validate() error {
	if err := errs.Check(app); err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("validate: %w", err))
	}
	return nil
}
