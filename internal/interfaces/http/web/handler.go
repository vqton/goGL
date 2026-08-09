package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

type module struct {
	Name string
	Slug string
}

var modules = []module{
	{"Cash", "cash"},
	{"Bank", "bank"},
	{"Purchase", "purchase"},
	{"Sales", "sales"},
	{"Invoice", "invoice"},
	{"Inventory", "inventory"},
	{"Tools", "tools"},
	{"Fixed Asset", "fixedasset"},
	{"Tax", "tax"},
	{"Payroll", "payroll"},
	{"Costing", "costing"},
	{"Ledger", "ledger"},
	{"Contract", "contract"},
	{"Budget", "budget"},
	{"Reporting", "reporting"},
	{"Setup", "setup"},
	{"Master Data", "masterdata"},
	{"User", "user"},
	{"System", "system"},
	{"Options", "options"},
	{"Document", "document"},
	{"Task", "task"},
	{"Audit", "audit"},
	{"Backup", "backup"},
}

func (h *Handler) Register(r *gin.Engine) {
	r.LoadHTMLGlob("web/templates/*.html")
	r.Static("/static", "web/static")
	r.GET("/", h.index)
}

func (h *Handler) index(c *gin.Context) {
	c.HTML(http.StatusOK, "base", gin.H{"Modules": modules})
}
