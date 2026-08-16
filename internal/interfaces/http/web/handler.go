package web

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"

	httpcashweb "goGL/internal/interfaces/http/webcash"
	httpwebledger "goGL/internal/interfaces/http/webledger"
	httpwebsetup "goGL/internal/interfaces/http/websetup"
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
	tmpl := template.Must(template.New("").Funcs(httpcashweb.Funcs()).Funcs(httpwebledger.Funcs()).Funcs(httpwebsetup.Funcs()).ParseGlob("web/templates/*.html"))
	template.Must(tmpl.ParseGlob("web/templates/cash/*.html"))
	template.Must(tmpl.ParseGlob("web/templates/ledger/*.html"))
	template.Must(tmpl.ParseGlob("web/templates/setup/*.html"))
	r.SetHTMLTemplate(tmpl)
	r.Static("/static", "web/static")
	r.GET("/", h.index)
}

func (h *Handler) index(c *gin.Context) {
	c.HTML(http.StatusOK, "base", gin.H{"Modules": modules})
}
