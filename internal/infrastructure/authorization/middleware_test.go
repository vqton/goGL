package authorization_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"goGL/internal/infrastructure/authorization"
)

func TestMiddleware_AllowsRoleMember(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if _, err := e.AddPolicy("role:viewer", "/api/v1/cash/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if _, err := e.AddGroupingPolicy("alice", "role:viewer"); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(authorization.AuthorizationMiddleware(e, authorization.HeaderPrincipalResolver("X-User-Id")))
	v1.GET("/cash/vouchers/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	v1.POST("/cash/vouchers", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cash/vouchers/1", nil)
	req.Header.Set("X-User-Id", "alice")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected alice GET to be allowed, got %d", rec.Code)
	}
}

func TestMiddleware_DeniesForbiddenAction(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if _, err := e.AddPolicy("role:viewer", "/api/v1/cash/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if _, err := e.AddGroupingPolicy("alice", "role:viewer"); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(authorization.AuthorizationMiddleware(e, authorization.HeaderPrincipalResolver("X-User-Id")))
	v1.GET("/cash/vouchers/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	v1.POST("/cash/vouchers", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cash/vouchers", nil)
	req.Header.Set("X-User-Id", "alice")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected alice POST to be denied (403), got %d", rec.Code)
	}
}

func TestMiddleware_AdminWildcard(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(authorization.AuthorizationMiddleware(e, authorization.HeaderPrincipalResolver("X-User-Id")))
	v1.GET("/cash/vouchers/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	v1.POST("/cash/vouchers", func(c *gin.Context) { c.Status(http.StatusCreated) })

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/api/v1/cash/vouchers", nil)
		req.Header.Set("X-User-Id", "admin")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code == http.StatusForbidden {
			t.Fatalf("expected admin %s to be allowed, got %d", method, rec.Code)
		}
	}
}

func TestMiddleware_AnonymousDenied(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if _, err := e.AddPolicy("role:viewer", "/api/v1/cash/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(authorization.AuthorizationMiddleware(e, authorization.HeaderPrincipalResolver("X-User-Id")))
	v1.GET("/cash/vouchers/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	// No identity header -> anonymous subject, fail closed.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cash/vouchers/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected anonymous to be denied (403), got %d", rec.Code)
	}

	// The 403 body must be a JSON error, not a leak of internals.
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error body, got: %q", rec.Body.String())
	}
	if body["error"] == nil {
		t.Fatal("expected error envelope in body")
	}
}

func TestMiddleware_UnmatchedRoutePassesThrough(t *testing.T) {
	db := openTestDB(t)
	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(authorization.AuthorizationMiddleware(e, authorization.HeaderPrincipalResolver("X-User-Id")))
	v1.GET("/cash/vouchers/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	// A path with no route handler: middleware must not panic; router 404s.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched route, got %d", rec.Code)
	}
}
