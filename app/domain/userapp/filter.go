package userapp

import (
	"net/http"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
	"github.com/jcpaschoal/spi-exata/business/types/name"
)

// queryParams struct interna para capturar os dados crus da URL.
type queryParams struct {
	Page             string
	Rows             string
	OrderBy          string
	ID               string
	Name             string
	Email            string
	StartCreatedDate string
	EndCreatedDate   string
}

// parseQueryParams extrai os parâmetros da request.
func parseQueryParams(r *http.Request) queryParams {
	values := r.URL.Query()

	return queryParams{
		Page:             values.Get("page"),
		Rows:             values.Get("rows"),
		OrderBy:          values.Get("orderBy"),
		ID:               values.Get("user_id"),
		Name:             values.Get("name"),
		Email:            values.Get("email"),
		StartCreatedDate: values.Get("start_created_date"),
		EndCreatedDate:   values.Get("end_created_date"),
	}
}

// parseFilter valida e converte os parâmetros crus para o filtro de domínio.
// Retorna erro agregado (FieldErrors) se houver falhas de validação.
func parseFilter(qp queryParams) (userbus.QueryFilter, error) {
	var fieldErrors errs.FieldErrors
	var filter userbus.QueryFilter

	if qp.ID != "" {
		id, err := uuid.Parse(qp.ID)
		switch err {
		case nil:
			filter.ID = &id
		default:
			fieldErrors.Add("user_id", err)
		}
	}

	if qp.Name != "" {
		nme, err := name.Parse(qp.Name)
		switch err {
		case nil:
			filter.Name = &nme
		default:
			fieldErrors.Add("name", err)
		}
	}

	if qp.Email != "" {
		addr, err := mail.ParseAddress(qp.Email)
		switch err {
		case nil:
			filter.Email = addr
		default:
			fieldErrors.Add("email", err)
		}
	}

	// Atenção: Mapeado para StartCreatedAt (conforme userbus/filter.go)
	if qp.StartCreatedDate != "" {
		t, err := time.Parse(time.RFC3339, qp.StartCreatedDate)
		switch err {
		case nil:
			filter.StartCreatedAt = &t
		default:
			fieldErrors.Add("start_created_date", err)
		}
	}

	// Atenção: Mapeado para EndCreatedAt (conforme userbus/filter.go)
	if qp.EndCreatedDate != "" {
		t, err := time.Parse(time.RFC3339, qp.EndCreatedDate)
		switch err {
		case nil:
			filter.EndCreatedAt = &t
		default:
			fieldErrors.Add("end_created_date", err)
		}
	}

	if fieldErrors != nil {
		return userbus.QueryFilter{}, fieldErrors.ToError()
	}

	return filter, nil
}
