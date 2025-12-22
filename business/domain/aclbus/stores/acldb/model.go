package acldb

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/domain/aclbus"
	"github.com/jcpaschoal/spi-exata/business/sdk/sqldb/dbarray"
	"github.com/jcpaschoal/spi-exata/business/types/actions"
	"github.com/jcpaschoal/spi-exata/business/types/resource"
)

type aclDB struct {
	ResourceID   uuid.UUID      `db:"resource_id"`
	UserID       uuid.UUID      `db:"user_id"`
	ResourceType string         `db:"resource_type"`
	Actions      dbarray.String `db:"actions"`
}

func toDBACL(userID uuid.UUID, resourceID uuid.UUID, res resource.Resource, acts []actions.Action) aclDB {
	actsStr := make([]string, len(acts))
	for i, act := range acts {
		actsStr[i] = act.String()
	}

	return aclDB{
		ResourceID:   resourceID,
		UserID:       userID,
		ResourceType: res.String(),
		Actions:      actsStr,
	}
}

func toBusACL(db aclDB) (aclbus.ACL, error) {
	res, err := resource.Parse(db.ResourceType)
	if err != nil {
		return aclbus.ACL{}, fmt.Errorf("parse resource: %w", err)
	}

	acts := make([]actions.Action, len(db.Actions))
	for i, actStr := range db.Actions {
		act, err := actions.Parse(actStr)
		if err != nil {
			return aclbus.ACL{}, fmt.Errorf("parse action: %w", err)
		}
		acts[i] = act
	}

	bus := aclbus.ACL{
		UserID:     db.UserID,
		ResourceID: db.ResourceID,
		Resource:   res,
		Actions:    acts,
	}

	return bus, nil
}

func toBusACLs(dbs []aclDB) ([]aclbus.ACL, error) {
	bus := make([]aclbus.ACL, len(dbs))

	for i, db := range dbs {
		var err error
		bus[i], err = toBusACL(db)
		if err != nil {
			return nil, err
		}
	}

	return bus, nil
}
