package dashboardapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/app/sdk/errs"
	"github.com/jcpaschoal/spi-exata/business/domain/dashboardbus"
	"github.com/jcpaschoal/spi-exata/business/types/name"
)

// Dashboard represents the application model for a dashboard.
type Dashboard struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenantId"`
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	Logo      []byte `json:"logo,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Encode implements the web.Encoder interface.
func (d Dashboard) Encode() ([]byte, string, error) {
	data, err := json.Marshal(d)
	return data, "application/json", err
}

func toAppDashboard(bus dashboardbus.Dashboard) Dashboard {
	domain := ""
	if bus.Domain != nil {
		domain = *bus.Domain
	}

	return Dashboard{
		ID:        bus.ID.String(),
		TenantID:  bus.TenantID.String(),
		Name:      bus.Name.String(),
		Domain:    domain,
		Logo:      bus.Logo,
		CreatedAt: bus.CreatedAt.Format(time.RFC3339),
		UpdatedAt: bus.UpdatedAt.Format(time.RFC3339),
	}
}

type NewDashboard struct {
	TenantID string `json:"tenantId" validate:"required,uuid"`
	Name     string `json:"name" validate:"required,min=3"`
	Domain   string `json:"domain" validate:"omitempty,hostname"`
	Logo     []byte `json:"logo"`
}

// Decode implements the web.Decoder interface.
func (app *NewDashboard) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

// Validate checks the data in the model is considered clean.
func (app NewDashboard) Validate() error {
	if err := errs.Check(app); err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("validate: %w", err))
	}
	return nil
}

func toBusNewDashboard(app NewDashboard) (dashboardbus.NewDashboard, error) {
	tenantID, err := uuid.Parse(app.TenantID)
	if err != nil {
		return dashboardbus.NewDashboard{}, fmt.Errorf("parse tenantID: %w", err)
	}

	n, err := name.Parse(app.Name)
	if err != nil {
		return dashboardbus.NewDashboard{}, fmt.Errorf("parse name: %w", err)
	}

	var domain *string
	if app.Domain != "" {
		domain = &app.Domain
	}

	return dashboardbus.NewDashboard{
		TenantID: tenantID,
		Name:     n,
		Domain:   domain,
		Logo:     app.Logo,
	}, nil
}

// =============================================================================

// UpdateDashboard defines the data needed to update a dashboard.
type UpdateDashboard struct {
	Name   *string `json:"name" validate:"omitempty,min=3"`
	Domain *string `json:"domain" validate:"omitempty,hostname"`
	Logo   []byte  `json:"logo" validate:"omitempty"`
}

// Decode implements the web.Decoder interface.
func (app *UpdateDashboard) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

// Validate checks the data in the model is considered clean.
func (app UpdateDashboard) Validate() error {
	if err := errs.Check(app); err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("validate: %w", err))
	}
	return nil
}

func toBusUpdateDashboard(app UpdateDashboard) (dashboardbus.UpdateDashboard, error) {
	var n *name.Name
	if app.Name != nil {
		parsedName, err := name.Parse(*app.Name)
		if err != nil {
			return dashboardbus.UpdateDashboard{}, fmt.Errorf("parse name: %w", err)
		}
		n = &parsedName
	}

	return dashboardbus.UpdateDashboard{
		Name:   n,
		Domain: app.Domain,
		Logo:   app.Logo,
	}, nil
}
