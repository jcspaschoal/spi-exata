package mid

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jcpaschoal/spi-exata/app/sdk/auth"
	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/sdk/web"
	"github.com/jcpaschoal/spi-exata/business/types/actions"
	"github.com/jcpaschoal/spi-exata/business/types/resource"
)

var ErrInvalidID = errors.New("ID is not in its proper form")

// Authorize valida se o usuário autenticado tem permissão para acessar o recurso.
// idKey: O nome do parâmetro na rota que contém o ID do recurso (ex: "id").
//
//	Se vazio "", o middleware assume que é uma ação de coleção (apenas check funcional).
func Authorize(ath *auth.Auth, res resource.Resource, idKey string) web.MidFunc {
	m := func(next web.HandlerFunc) web.HandlerFunc {
		h := func(ctx context.Context, r *http.Request) web.Encoder {

			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			userID, err := GetUserID(ctx)
			if err != nil {
				return errs.New(errs.Unauthenticated, err)
			}

			act, err := mapHTTPMethodToAction(r.Method)
			if err != nil {
				return errs.New(errs.FailedPrecondition, err)
			}

			if idKey != "" {
				idKey = web.Param(r, idKey)

			}

			if err := ath.Authorize(userID, res, act, idKey); err != nil {
				return errs.New(errs.PermissionDenied, err)
			}

			return next(ctx, r)
		}

		return h
	}

	return m
}

func mapHTTPMethodToAction(method string) (actions.Action, error) {
	switch method {
	case http.MethodGet:
		return actions.Get, nil
	case http.MethodPost:
		return actions.Create, nil
	case http.MethodPut, http.MethodPatch:
		return actions.Update, nil
	case http.MethodDelete:
		return actions.Delete, nil
	default:
		return actions.Action{}, fmt.Errorf("action: %s", method)
	}
}
