package middleware

import "github.com/gin-gonic/gin"

const (
	TenantHeader = "X-Tenant-ID"
	TenantKey    = "tenantID"
)

func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader(TenantHeader)
		if tenantID == "" {
			tenantID = "tenant_demo"
		}

		c.Set(TenantKey, tenantID)
		c.Next()
	}
}

func TenantID(c *gin.Context) string {
	value, ok := c.Get(TenantKey)
	if !ok {
		return "tenant_demo"
	}

	tenantID, ok := value.(string)
	if !ok || tenantID == "" {
		return "tenant_demo"
	}

	return tenantID
}
