package userbus

import "github.com/jcpaschoal/spi-exata/business/sdk/order"

var DefaultOrderBy = order.NewBy(OrderByID, order.ASC)

const (
	OrderByID      = "a"
	OrderByName    = "b"
	OrderByEmail   = "c"
	OrderByRole    = "d"
	OrderByEnabled = "e"
)
