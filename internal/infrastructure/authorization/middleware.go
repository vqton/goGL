package authorization

import (
	"net/http"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

// AnonymousSubject is the subject used when no authenticated principal is
// available. The system fails closed: anonymous is only granted what a policy
// explicitly allows.
const AnonymousSubject = "anonymous"

// PrincipalResolver resolves the current subject (user id) for a request.
// Swap in a session/cookie/token resolver when authentication lands.
type PrincipalResolver func(*gin.Context) string

// HeaderPrincipalResolver reads the subject from the named request header.
// It is a development seam until real authentication is implemented: anyone
// can set the header, so it must not be used as the identity boundary in
// production.
func HeaderPrincipalResolver(header string) PrincipalResolver {
	return func(c *gin.Context) string {
		if sub := c.GetHeader(header); sub != "" {
			return sub
		}
		return AnonymousSubject
	}
}

// AuthorizationMiddleware enforces casbin policies for every request on the
// group it is mounted on. The enforcement triple is:
//
//	sub = resolved principal (user id)
//	obj = matched route pattern, e.g. /api/v1/cash/vouchers/:id
//	act = HTTP method
//
// Requests for paths without a matched route are passed through so the router
// answers with 404 rather than the middleware masking it as 403.
func AuthorizationMiddleware(e *casbin.Enforcer, resolve PrincipalResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		obj := c.FullPath()
		if obj == "" {
			c.Next()
			return
		}

		allowed, err := e.Enforce(resolve(c), obj, c.Request.Method)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "AUTHZ_ERROR",
					"message": "authorization check failed",
				},
			})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "you are not allowed to access this resource",
				},
			})
			return
		}
		c.Next()
	}
}
