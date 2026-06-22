package service

import (
	"errors"
	"testing"

	"geopress/backend/internal/domain"
)

func TestListResponseNormalizesNilItems(t *testing.T) {
	response := NewListResponse[string](nil, 0, PageRequest{Page: -1, PageSize: 1000})

	if response.Items == nil {
		t.Fatalf("items should be an empty slice, not nil")
	}
	if response.Page != 1 {
		t.Fatalf("page = %d, want 1", response.Page)
	}
	if response.PageSize != MaxPageSize {
		t.Fatalf("page size = %d, want %d", response.PageSize, MaxPageSize)
	}
}

func TestRequestContextPermissionGuards(t *testing.T) {
	viewer := RequestContext{Workspace: WorkspaceContext{WorkspaceID: "wks", Role: domain.WorkspaceRoleViewer}}
	if err := viewer.RequireWorkspaceEditor(); Code(err) != ErrorCodeForbidden {
		t.Fatalf("viewer editor error code = %s, want forbidden", Code(err))
	}

	editor := RequestContext{Workspace: WorkspaceContext{WorkspaceID: "wks", Role: domain.WorkspaceRoleEditor}}
	if err := editor.RequireWorkspaceEditor(); err != nil {
		t.Fatalf("editor should pass workspace edit guard: %v", err)
	}

	platformAdmin := RequestContext{Actor: ActorContext{IsPlatformAdmin: true}}
	if err := platformAdmin.RequirePlatformAdmin(); err != nil {
		t.Fatalf("platform admin should pass guard: %v", err)
	}
}

func TestServiceErrorSafeMessage(t *testing.T) {
	cause := errors.New("sql: secret detail")
	err := WrapError(ErrorCodeDependency, "dependency failed", cause)

	if !errors.Is(err, cause) {
		t.Fatalf("wrapped error should preserve cause")
	}
	if Code(err) != ErrorCodeDependency {
		t.Fatalf("code = %s, want dependency", Code(err))
	}
	if SafeMessage(err) != "dependency failed" {
		t.Fatalf("safe message = %q, want public service message", SafeMessage(err))
	}
	if SafeMessage(cause) != "internal server error" {
		t.Fatalf("plain error should not leak internal details")
	}
}
