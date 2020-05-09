package iam

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/vault/sdk/helper/base62"
	"github.com/hashicorp/watchtower/internal/db"
	"github.com/hashicorp/watchtower/internal/iam/store"
	"google.golang.org/protobuf/proto"
)

// RoleGrant defines the grants that are assigned to a role
type RoleGrant struct {
	*store.RoleGrant
	tableName string `gorm:"-"`
}

// ensure that RoleGrant implements the interfaces of: Resource, ClonableResource and db.VetForWriter
var _ Resource = (*RoleGrant)(nil)
var _ Clonable = (*RoleGrant)(nil)
var _ db.VetForWriter = (*RoleGrant)(nil)

// NewRoleGrant creates a new grant with a scope (project/organization)
// options include: WithName
func NewRoleGrant(scope *Scope, role *Role, grant string, opt ...Option) (*RoleGrant, error) {
	opts := getOpts(opt...)
	withName := opts.withName
	if scope == nil {
		return nil, errors.New("error the role grant scope is nil")
	}
	if scope.Type != OrganizationScope.String() &&
		scope.Type != ProjectScope.String() {
		return nil, errors.New("role grants can only be within an organization or project scope")
	}
	if role == nil {
		return nil, errors.New("error role is nil")
	}
	if role.PublicId == "" {
		return nil, errors.New("error role id is unset")
	}
	publicId, err := base62.Random(20)
	if err != nil {
		return nil, fmt.Errorf("error generating public id %w for new role grant", err)
	}
	rg := &RoleGrant{
		RoleGrant: &store.RoleGrant{
			PublicId: publicId,
			ScopeId:  scope.GetPublicId(),
			RoleId:   role.PublicId,
			Grant:    grant,
		},
	}
	if withName != "" {
		rg.Name = withName
	}
	return rg, nil
}

func allocRoleGrant() RoleGrant {
	return RoleGrant{
		RoleGrant: &store.RoleGrant{},
	}
}

// Clone creates a clone of the RoleGrant
func (g *RoleGrant) Clone() interface{} {
	cp := proto.Clone(g.RoleGrant)
	return &RoleGrant{
		RoleGrant: cp.(*store.RoleGrant),
	}
}

// VetForWrite implements db.VetForWrite() interface
func (g *RoleGrant) VetForWrite(ctx context.Context, r db.Reader, opType db.OpType, opt ...db.Option) error {
	if g.PublicId == "" {
		return errors.New("error public id is empty string for grant write")
	}
	if g.ScopeId == "" {
		return errors.New("error scope id not set for grant write")
	}
	// make sure the scope is valid for users
	if err := g.scopeIsValid(ctx, r); err != nil {
		return err
	}
	return nil
}

func (g *RoleGrant) scopeIsValid(ctx context.Context, r db.Reader) error {
	ps, err := LookupScope(ctx, r, g)
	if err != nil {
		return err
	}
	if ps.Type != OrganizationScope.String() && ps.Type != ProjectScope.String() {
		return errors.New("error scope is not an organization or project for the grant")
	}
	return nil
}

// GetScope returns the scope for the RoleGrant
func (g *RoleGrant) GetScope(ctx context.Context, r db.Reader) (*Scope, error) {
	return LookupScope(ctx, r, g)
}

// ResourceType returns the type of the RoleGrant
func (*RoleGrant) ResourceType() ResourceType { return ResourceTypeRoleGrant }

// Actions returns the  available actions for RoleGrant
func (*RoleGrant) Actions() map[string]Action {
	return CrudActions()
}

// TableName returns the tablename to override the default gorm table name
func (g *RoleGrant) TableName() string {
	if g.tableName != "" {
		return g.tableName
	}
	return "iam_role_grant"
}

// SetTableName sets the tablename and satisfies the ReplayableMessage interface
func (g *RoleGrant) SetTableName(n string) {
	if n != "" {
		g.tableName = n
	}
}
