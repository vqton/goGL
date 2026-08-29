package authorization

import (
	"context"
	"net/http"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

// AnonymousSubject is the subject used when no authenticated principal is
// available. The system fails closed: anonymous is only granted what a policy
// explicitly allows.
const AnonymousSubject = "anonymous"

// PrincipalResolver resolves the current subject (user id) for a request.
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

// SessionValidator resolves a session token to a subject username. Satisfied
// by the auth application service through a small adapter (auth.Validate
// returns the full user).
type SessionValidator interface {
	Validate(ctx context.Context, token string) (string, error)
}

// SessionValidatorFunc adapts a function to SessionValidator.
type SessionValidatorFunc func(ctx context.Context, token string) (string, error)

func (f SessionValidatorFunc) Validate(ctx context.Context, token string) (string, error) {
	return f(ctx, token)
}

// SessionPrincipalResolver resolves the subject from the session cookie.
// Anonymous requests (missing/invalid cookie) fail closed to AnonymousSubject.
func SessionPrincipalResolver(cookieName string, validate SessionValidator) PrincipalResolver {
	return func(c *gin.Context) string {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			return AnonymousSubject
		}
		sub, err := validate.Validate(c.Request.Context(), token)
		if err != nil || sub == "" {
			return AnonymousSubject
		}
		return sub
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
