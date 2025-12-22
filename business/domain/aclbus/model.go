package aclbus

import (
	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/types/actions"
	"github.com/jcpaschoal/spi-exata/business/types/resource"
)

// ACL represents a persisted access control rule.
type ACL struct {
	UserID     uuid.UUID
	ResourceID uuid.UUID
	Resource   resource.Resource
	Actions    []actions.Action
}

// NewACL contains information needed to create a new ACL Rule.
type NewACL struct {
	ResourceID uuid.UUID
	Resource   resource.Resource
	Actions    []actions.Action
}

// ACLUpdate contains information needed to update an ACL Rule.
type ACLUpdate struct {
	ResourceID uuid.UUID
	Resource   resource.Resource // Direct value required to identify resource type in SQL
	Actions    []actions.Action
}

// AccessCheck represents a request to validate permission for a SPECIFIC INSTANCE.
type AccessCheck struct {
	UserID     uuid.UUID
	ResourceID uuid.UUID
	Resource   resource.Resource
	Action     actions.Action
}

// ResourceCheck represents a request to validate permission for a RESOURCE TYPE.
type ResourceCheck struct {
	UserID   uuid.UUID
	Resource resource.Resource // Identifies the Type (e.g., Dashboard, Page)
	Action   actions.Action
}
