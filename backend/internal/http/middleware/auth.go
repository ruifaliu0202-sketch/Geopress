package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	UserIDKey           = "userID"
	WorkspaceHeader     = "X-Workspace-ID"
	WorkspaceIDKey      = "workspaceID"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader(AuthorizationHeader)
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		userID, ok := tokenUserID(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid bearer token"})
			return
		}

		workspaceID := c.GetHeader(WorkspaceHeader)
		c.Set(UserIDKey, userID)
		c.Set(WorkspaceIDKey, workspaceID)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) string {
	value, ok := c.Get(UserIDKey)
	if !ok {
		return ""
	}
	userID, _ := value.(string)
	return userID
}

func CurrentWorkspaceID(c *gin.Context) string {
	value, ok := c.Get(WorkspaceIDKey)
	if !ok {
		return ""
	}
	workspaceID, _ := value.(string)
	return workspaceID
}

func tokenUserID(token string) (string, bool) {
	switch token {
	case "demo-token":
		return "usr_demo", true
	case "growth-token":
		return "usr_growth", true
	default:
		return "", false
	}
}
