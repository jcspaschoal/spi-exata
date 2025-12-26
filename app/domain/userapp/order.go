package userapp

import (
	"github.com/jcpaschoal/spi-exata/business/domain/userbus"
)

var orderByFields = map[string]string{
	"user_id": userbus.OrderByID,
	"name":    userbus.OrderByName,
	"email":   userbus.OrderByEmail,
	"role":    userbus.OrderByRole,
	"enabled": userbus.OrderByEnabled,
}
