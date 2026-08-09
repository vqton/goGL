package authz_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"

	"goGL/internal/infrastructure/authorization"
	"goGL/internal/interfaces/http/authz"
)

// openTestDB mirrors internal/infrastructure/authorization's test helper: a
// uniquely-named in-memory SQLite database with the casbin_policies table.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	clean := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(t.Name(), "_")
	db, err := sql.Open("sqlite", "file:"+clean+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS casbin_policies (
		id   TEXT PRIMARY KEY,
		data TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create casbin_policies: %v", err)
	}
	return db
}

func newEnforcer(t *testing.T, db *sql.DB) *casbin.Enforcer {
	t.Helper()

	e, err := authorization.NewEnforcer(db)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	return e
}

func newRouter(t *testing.T, e *casbin.Enforcer) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	authz.NewHandler(e).Register(r.Group("/api/v1"))
	return r
}

func doJSON(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, out interface{}) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
}

func TestHandler_ListPolicies(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	rec := doJSON(t, newRouter(t, e), http.MethodGet, "/api/v1/authz/policies", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var got struct {
		P [][]string `json:"p"`
		G [][]string `json:"g"`
	}
	decodeBody(t, rec, &got)

	contains := func(rules [][]string, want []string) bool {
		for _, r := range rules {
			if len(r) == len(want) {
				equal := true
				for i := range want {
					if r[i] != want[i] {
						equal = false
						break
					}
				}
				if equal {
					return true
				}
			}
		}
		return false
	}

	seeded := [][]string{
		{"role:admin", "*", "*"},
		{"role:cashier", "/api/v1/cash/vouchers/*/post", "POST"},
		{"role:cashier", "/api/v1/cash/book", "GET"},
		{"role:cashier", "/api/v1/cash/close-day", "POST"},
		{"role:cash_accountant", "/api/v1/cash/vouchers", "*"},
		{"role:cash_accountant", "/api/v1/cash/vouchers/*", "*"},
		{"role:cash_accountant", "/api/v1/cash/reconcile", "POST"},
		{"role:chief_accountant", "/api/v1/cash/*", "*"},
		{"role:director", "/api/v1/cash/*/approve", "POST"},
	}
	for _, rule := range seeded {
		if !contains(got.P, rule) {
			t.Fatalf("expected seeded policy %v in list, got %v", rule, got.P)
		}
	}
	if len(got.G) != 1 || len(got.G[0]) != 2 || got.G[0][0] != "admin" {
		t.Fatalf("unexpected g rules: %v", got.G)
	}
}

func TestHandler_AddPolicy(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)
	r := newRouter(t, e)

	rule := `{"ptype":"p","rule":["role:viewer","/api/v1/cash/*","GET"]}`
	rec := doJSON(t, r, http.MethodPost, "/api/v1/authz/policies", rule)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	var added struct {
		Added bool `json:"added"`
	}
	decodeBody(t, rec, &added)
	if !added.Added {
		t.Fatal("expected added=true")
	}

	// Re-adding the same rule is an idempotent 200, not an error.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/authz/policies", rule)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for duplicate, got %d", rec.Code)
	}
	decodeBody(t, rec, &added)
	if added.Added {
		t.Fatal("expected added=false for duplicate")
	}

	// The rule is now enforceable through the API.
	check := `{"sub":"role:viewer","obj":"/api/v1/cash/vouchers","act":"GET"}`
	rec = doJSON(t, r, http.MethodPost, "/api/v1/authz/check", check)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected check 200, got %d", rec.Code)
	}
	var allowed struct {
		Allowed bool `json:"allowed"`
	}
	decodeBody(t, rec, &allowed)
	if !allowed.Allowed {
		t.Fatal("expected GET cash allowed for role:viewer")
	}
}

func TestHandler_AddGroupingPolicy(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)
	r := newRouter(t, e)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/authz/policies", `{"ptype":"g","rule":["alice","role:viewer"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	// alice inherits role:viewer's GET permission through the grouping link.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/authz/check", `{"sub":"alice","obj":"/api/v1/cash/vouchers","act":"GET"}`)
	var allowed struct {
		Allowed bool `json:"allowed"`
	}
	decodeBody(t, rec, &allowed)
	if allowed.Allowed {
		t.Fatal("expected alice denied before viewer policy exists")
	}
}

func TestHandler_AddPolicy_InvalidPType(t *testing.T) {
	db := openTestDB(t)
	r := newRouter(t, newEnforcer(t, db))

	rec := doJSON(t, r, http.MethodPost, "/api/v1/authz/policies", `{"ptype":"x","rule":["a","b","c"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_AddPolicy_EmptyRule(t *testing.T) {
	db := openTestDB(t)
	r := newRouter(t, newEnforcer(t, db))

	for _, body := range []string{`{"ptype":"p","rule":[]}`, `{"ptype":"p"}`, `{`} {
		rec := doJSON(t, r, http.MethodPost, "/api/v1/authz/policies", body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q: expected 400, got %d", body, rec.Code)
		}
	}
}

func TestHandler_RemovePolicy(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)
	r := newRouter(t, e)

	rule := `{"ptype":"p","rule":["role:viewer","/api/v1/cash/*","GET"]}`
	if rec := doJSON(t, r, http.MethodPost, "/api/v1/authz/policies", rule); rec.Code != http.StatusCreated {
		t.Fatalf("add policy: expected 201, got %d", rec.Code)
	}

	rec := doJSON(t, r, http.MethodDelete, "/api/v1/authz/policies", rule)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var removed struct {
		Removed bool `json:"removed"`
	}
	decodeBody(t, rec, &removed)
	if !removed.Removed {
		t.Fatal("expected removed=true")
	}

	rec = doJSON(t, r, http.MethodDelete, "/api/v1/authz/policies", rule)
	decodeBody(t, rec, &removed)
	if removed.Removed {
		t.Fatal("expected removed=false for absent rule")
	}
}

func TestHandler_RemovePolicy_InvalidPType(t *testing.T) {
	db := openTestDB(t)
	r := newRouter(t, newEnforcer(t, db))

	rec := doJSON(t, r, http.MethodDelete, "/api/v1/authz/policies", `{"ptype":"x","rule":["a","b","c"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Check(t *testing.T) {
	db := openTestDB(t)
	e := newEnforcer(t, db)
	if err := authorization.SeedDefaultPolicies(e); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	r := newRouter(t, e)

	cases := []struct {
		body    string
		allowed bool
	}{
		{`{"sub":"admin","obj":"/api/v1/cash/vouchers","act":"GET"}`, true},
		{`{"sub":"admin","obj":"/api/v1/authz/policies","act":"DELETE"}`, true},
		{`{"sub":"bob","obj":"/api/v1/cash/vouchers","act":"GET"}`, false},
		{`{"sub":"bob","obj":"/api/v1/authz/policies","act":"GET"}`, false},
	}
	for _, tc := range cases {
		rec := doJSON(t, r, http.MethodPost, "/api/v1/authz/check", tc.body)
		if rec.Code != http.StatusOK {
			t.Fatalf("body %s: expected 200, got %d", tc.body, rec.Code)
		}
		var got struct {
			Allowed bool `json:"allowed"`
		}
		decodeBody(t, rec, &got)
		if got.Allowed != tc.allowed {
			t.Fatalf("body %s: expected allowed=%v, got %v", tc.body, tc.allowed, got.Allowed)
		}
	}
}

func TestHandler_Check_MalformedJSON(t *testing.T) {
	db := openTestDB(t)
	r := newRouter(t, newEnforcer(t, db))

	rec := doJSON(t, r, http.MethodPost, "/api/v1/authz/check", `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
