package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"geopress/backend/internal/database"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	UserIDKey           = "userID"
	WorkspaceHeader     = "X-Workspace-ID"
	WorkspaceIDKey      = "workspaceID"
)

type TokenResolver func(ctx context.Context, token string) (string, bool, error)

func Auth() gin.HandlerFunc {
	return AuthWithTokenResolver(nil)
}

func AuthWithDatabase(db *database.DB) gin.HandlerFunc {
	return AuthWithTokenResolver(func(ctx context.Context, token string) (string, bool, error) {
		return databaseTokenUserID(ctx, db, token)
	})
}

func AuthWithTokenResolver(resolve TokenResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader(AuthorizationHeader)
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		userID, ok := tokenUserID(token)
		if !ok {
			if resolve == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid bearer token"})
				return
			}
			var err error
			userID, ok, err = resolve(c.Request.Context(), token)
			if !ok {
				if err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session token lookup failed"})
					return
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid bearer token"})
				return
			}
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session token lookup failed"})
				return
			}
		}

		workspaceID := c.GetHeader(WorkspaceHeader)
		c.Set(UserIDKey, userID)
		c.Set(WorkspaceIDKey, workspaceID)
		c.Next()
	}
}

func databaseTokenUserID(ctx context.Context, db *database.DB, token string) (string, bool, error) {
	if db == nil || db.SQL() == nil {
		return "", false, nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return db.UserIDBySessionToken(dbCtx, token)
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
