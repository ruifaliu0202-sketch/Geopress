package service

import "geopress/backend/internal/domain"

type ActorContext struct {
	UserID          string
	IsPlatformAdmin bool
}

type WorkspaceContext struct {
	WorkspaceID string
	TenantID    string
	Role        domain.WorkspaceRole
}

type RequestContext struct {
	Actor     ActorContext
	Workspace WorkspaceContext
}

func (ctx RequestContext) RequireWorkspace() error {
	if ctx.Workspace.WorkspaceID == "" {
		return NewError(ErrorCodeValidation, "workspace is required")
	}
	return nil
}

func (ctx RequestContext) RequireWorkspaceEditor() error {
	if err := ctx.RequireWorkspace(); err != nil {
		return err
	}
	// 工作区内的写操作必须显式携带成员角色，避免后续服务绕过租户权限判断。
	if !ctx.Workspace.Role.CanEditWorkspaceContent() {
		return NewError(ErrorCodeForbidden, "workspace edit permission is required")
	}
	return nil
}

func (ctx RequestContext) RequireWorkspaceAdmin() error {
	if err := ctx.RequireWorkspace(); err != nil {
		return err
	}
	if !ctx.Workspace.Role.CanManageWorkspace() {
		return NewError(ErrorCodeForbidden, "workspace admin permission is required")
	}
	return nil
}

func (ctx RequestContext) RequirePlatformAdmin() error {
	if !ctx.Actor.IsPlatformAdmin {
		return NewError(ErrorCodeForbidden, "platform admin permission is required")
	}
	return nil
}
