package masterdata

import (
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/masterdata"
	md "goGL/internal/domain/masterdata"
)

// Handler exposes the master-data API and the web UI page. It is registered by
// the web package under BOTH /api/v1 and /web/master-data groups (see
// web.ServerConfig.MasterData), so every route below exists on both prefixes.
type Handler struct {
	svc masterdata.Service
}

// NewHandler builds the HTTP handler over the master-data service.
func NewHandler(svc masterdata.Service) *Handler {
	return &Handler{svc: svc}
}

// Register wires all master-data routes onto the given group.
func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/master-data")

	g.GET("/", h.pageIndex)
	g.GET("/catalogs", h.catalogs)
	g.POST("/import", h.importCSV)
	g.GET("/regime", h.getRegime)
	g.POST("/settings/regime", h.setRegime)
	g.POST("/accounts/seed", h.seedAccounts)

	g.GET("/:kind", h.listRecords)
	g.POST("/:kind", h.createRecord)
	g.GET("/:kind/export", h.exportCSV)
	g.GET("/:kind/:code", h.getRecord)
	g.PUT("/:kind/:code", h.updateRecord)
	g.DELETE("/:kind/:code", h.deleteRecord)
	g.POST("/:kind/:code/deactivate", h.deactivateRecord)
	g.POST("/:kind/:code/deactivate-force", h.forceDeactivateRecord)
	g.POST("/:kind/:code/activate", h.activateRecord)
	g.POST("/:kind/:code/references", h.setReferences)
	g.POST("/:kind/merge", h.mergeRecords)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := strings.TrimSpace(c.GetHeader("X-User-Id")); u != "" {
		return u
	}
	return "system"
}

func (h *Handler) kind(c *gin.Context) (md.Kind, bool) {
	k := md.Kind(c.Param("kind"))
	if !k.IsKind() {
		writeError(c, &md.ValidationError{Kind: k, MessageVn: "Loại danh mục không hỗ trợ",
			MessageEn: "unsupported kind: " + string(k)})
		return "", false
	}
	return k, true
}

func (h *Handler) pageIndex(c *gin.Context) {
	regime, _ := h.svc.GetRegime(c.Request.Context())
	c.HTML(http.StatusOK, "masterdata", gin.H{
		"Title":  "Danh mục cơ bản",
		"Actor":  h.actor(c),
		"Now":    time.Now().Format("02/01/2006 15:04"),
		"Regime": regime,
	})
}

func (h *Handler) catalogs(c *gin.Context) {
	type catalog struct {
		Kind   string `json:"kind"`
		Label  string `json:"label"`
		Prefix string `json:"prefix"`
		Auto   bool   `json:"auto_code"`
	}
	out := make([]catalog, 0, len(md.Kinds))
	for _, k := range md.Kinds {
		out = append(out, catalog{Kind: string(k), Label: k.Label(), Prefix: k.Prefix(), Auto: k.AutoCode()})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (h *Handler) createRecord(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	var in md.Record
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	rec, err := h.svc.Create(c.Request.Context(), kind, &in, h.actor(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": rec})
}

func (h *Handler) updateRecord(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	var patch md.Record
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	rec, err := h.svc.Update(c.Request.Context(), kind, c.Param("code"), &patch, h.actor(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rec})
}

func (h *Handler) getRecord(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	rec, err := h.svc.Get(c.Request.Context(), kind, c.Param("code"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rec})
}

func (h *Handler) listRecords(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	items, total, err := h.svc.List(c.Request.Context(), kind,
		c.Query("q"), c.Query("group"), c.Query("state"), page, size)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "total": total})
}

func (h *Handler) deactivateRecord(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	rec, err := h.svc.Deactivate(c.Request.Context(), kind, c.Param("code"), body.Reason, h.actor(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rec})
}

func (h *Handler) forceDeactivateRecord(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	rec, err := h.svc.ForceDeactivate(c.Request.Context(), kind, c.Param("code"), body.Reason, h.actor(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rec})
}

func (h *Handler) activateRecord(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	rec, err := h.svc.Activate(c.Request.Context(), kind, c.Param("code"), h.actor(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rec})
}

func (h *Handler) deleteRecord(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), kind, c.Param("code"), h.actor(c)); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) mergeRecords(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	var body struct {
		Keep   string   `json:"keep"`
		Dupes  []string `json:"dupes"`
		DryRun bool     `json:"dry_run"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	res, err := h.svc.Merge(c.Request.Context(), kind, body.Keep, body.Dupes, h.actor(c), body.DryRun)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *Handler) setReferences(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	var body struct {
		Count int64 `json:"count"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	if err := h.svc.SetReferenceCount(c.Request.Context(), kind, c.Param("code"), body.Count); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"count": body.Count}})
}

func (h *Handler) getRegime(c *gin.Context) {
	regime, err := h.svc.GetRegime(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"regime": regime}})
}

func (h *Handler) setRegime(c *gin.Context) {
	var body struct {
		Regime string `json:"regime"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return
	}
	if err := h.svc.SetRegime(c.Request.Context(), body.Regime, h.actor(c)); err != nil {
		writeError(c, err)
		return
	}
	regime, _ := h.svc.GetRegime(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"regime": regime}})
}

func (h *Handler) seedAccounts(c *gin.Context) {
	created, err := h.svc.SeedAccounts(c.Request.Context(), h.actor(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"created": created}})
}

// importCSV accepts a multipart upload (field "file") with optional "dry_run".
// Columns are header-mapped; "code"/"ma", "name"/"ten" aliases are supported.
func (h *Handler) importCSV(c *gin.Context) {
	kind := md.Kind(c.Query("kind"))
	if !kind.IsKind() {
		writeError(c, &md.ValidationError{Kind: kind, MessageVn: "Thiếu hoặc sai loại danh mục (kind)",
			MessageEn: "missing or invalid kind query param"})
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file field"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		writeError(c, err)
		return
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid csv: " + err.Error()})
		return
	}
	dryRun := strings.EqualFold(c.PostForm("dry_run"), "true") || c.PostForm("dry_run") == "1"
	res, err := h.svc.ImportRows(c.Request.Context(), kind, rows, h.actor(c), dryRun)
	if err != nil {
		writeError(c, err)
		return
	}
	if !dryRun && len(res.Errors) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": res})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res, "warnings": len(res.Errors) > 0})
}

func (h *Handler) exportCSV(c *gin.Context) {
	kind, ok := h.kind(c)
	if !ok {
		return
	}
	items, err := h.svc.Export(c.Request.Context(), kind,
		c.Query("q"), c.Query("group"), c.Query("state"))
	if err != nil {
		writeError(c, err)
		return
	}

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"kind", "code", "name", "name_en", "group_code",
		"state", "ref_count", "account_type", "allow_post", "valid_from", "valid_to"})
	for _, r := range items {
		_ = w.Write([]string{
			string(r.Kind), r.Code, r.Name, r.NameEN, r.GroupCode,
			string(r.State), strconv.FormatInt(r.RefCount, 10),
			string(r.AccountType), strconv.FormatBool(r.AllowPost),
			r.ValidFrom, r.ValidTo,
		})
	}
	w.Flush()

	c.Header("Content-Disposition", "attachment; filename="+string(kind)+".csv")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.String(http.StatusOK, sb.String())
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, md.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, md.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate code"})
	case errors.Is(err, md.ErrBlockedRefs):
		c.JSON(http.StatusConflict, gin.H{"error": "record is referenced"})
	case errors.Is(err, md.ErrInactive):
		c.JSON(http.StatusConflict, gin.H{"error": "invalid state transition"})
	case errors.Is(err, md.ErrCycle):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "group cycle"})
	case errors.Is(err, md.ErrCodeImmutable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "code is immutable"})
	case errors.Is(err, md.ErrUnknownKind):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		var ve *md.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":      ve.MessageEn,
				"message_vn": ve.MessageVn,
				"message_en": ve.MessageEn,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
