package aclbus

import (
	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/types/actions"
	"github.com/jcpaschoal/spi-exata/business/types/resource"
)

type ACL struct {
	UserID     uuid.UUID
	ResourceID uuid.UUID
	Resource   resource.Resource
	Actions    []actions.Action
}

type ACLUpdate struct {
	Resource *resource.Resource
	Actions  []actions.Action
}

type NewACL struct {
	Resource resource.Resource
	Actions  []actions.Action
}
