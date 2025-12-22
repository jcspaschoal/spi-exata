package aclcache

import (
	"context"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"github.com/jcpaschoal/spi-exata/business/types/actions"
	"github.com/jcpaschoal/spi-exata/business/types/resource"
	"github.com/jcpaschoal/spi-exata/business/types/role"
	"github.com/jcpaschoal/spi-exata/foundatiton/logger"
)

const casbinModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, "ROLE:ADMIN") || (g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act)
`

type memoryCache struct {
	log      *logger.Logger
	enforcer *casbin.Enforcer
}

func newMemoryCache(log *logger.Logger) (*memoryCache, error) {
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		return nil, fmt.Errorf("load model: %w", err)
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("create enforcer: %w", err)
	}

	return &memoryCache{
		log:      log,
		enforcer: e,
	}, nil
}

// add inserts a policy. Failures are logged with the provided context.
func (c *memoryCache) add(ctx context.Context, userID uuid.UUID, res resource.Resource, resID uuid.UUID, act actions.Action) {
	sub := userID.String()
	obj := c.resolveObject(res, resID)
	action := act.String()

	if _, err := c.enforcer.AddPolicy(sub, obj, action); err != nil {
		c.log.Error(ctx, "aclcache: casbin add policy failed", "sub", sub, "obj", obj, "act", action, "err", err)
	}
}

// clearInstanceRules removes rules for a specific ID. Failures are logged with context.
func (c *memoryCache) clearInstanceRules(ctx context.Context, userID uuid.UUID, resID uuid.UUID) {
	if resID == uuid.Nil {
		return
	}

	sub := userID.String()
	obj := resID.String()

	if _, err := c.enforcer.RemoveFilteredPolicy(0, sub, obj); err != nil {
		c.log.Error(ctx, "aclcache: casbin clear instance rules failed", "sub", sub, "obj", obj, "err", err)
	}
}

// clearResourceRules removes rules for a specific Type. Failures are logged with context.
func (c *memoryCache) clearResourceRules(ctx context.Context, userID uuid.UUID, res resource.Resource) {
	sub := userID.String()
	obj := res.String()

	if _, err := c.enforcer.RemoveFilteredPolicy(0, sub, obj); err != nil {
		c.log.Error(ctx, "aclcache: casbin clear resource rules failed", "sub", sub, "obj", obj, "err", err)
	}
}

// check validates permission using the context for potential logging/tracing internally if needed.
func (c *memoryCache) check(ctx context.Context, userID uuid.UUID, res resource.Resource, resID uuid.UUID, act actions.Action) error {
	sub := userID.String()
	action := act.String()

	// 1. Check Instance (Specific ID)
	if resID != uuid.Nil {
		ok, err := c.enforcer.Enforce(sub, resID.String(), action)
		if err != nil {
			return fmt.Errorf("enforce instance: %w", err)
		}
		if ok {
			return nil
		}
	}

	// 2. Check Functional (Role/Tag)
	ok, err := c.enforcer.Enforce(sub, res.String(), action)
	if err != nil {
		return fmt.Errorf("enforce tag: %w", err)
	}
	if ok {
		return nil
	}

	return fmt.Errorf("denied in cache")
}

// loadRoles populates roles using the context for logging failures.
func (c *memoryCache) loadRoles(ctx context.Context, userRoles map[uuid.UUID]role.Role) {
	for uid, r := range userRoles {
		c.setUserRole(ctx, uid, r)
	}
}

// setUserRole updates the role using the context for logging.
func (c *memoryCache) setUserRole(ctx context.Context, userID uuid.UUID, r role.Role) {
	sub := userID.String()
	roleName := "ROLE:" + strings.ToUpper(r.String())

	if _, err := c.enforcer.RemoveFilteredGroupingPolicy(0, sub); err != nil {
		c.log.Error(ctx, "aclcache: casbin failed to remove old role", "sub", sub, "err", err)
	}

	if _, err := c.enforcer.AddGroupingPolicy(sub, roleName); err != nil {
		c.log.Error(ctx, "aclcache: casbin failed to set new role", "sub", sub, "role", roleName, "err", err)
	}
}

func (c *memoryCache) resolveObject(res resource.Resource, id uuid.UUID) string {
	if id != uuid.Nil {
		return id.String()
	}
	return res.String()
}
