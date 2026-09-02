package inventory

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"goGL/internal/application/inventory"
	"goGL/internal/domain/core"
	domaininventory "goGL/internal/domain/inventory"
)

type Handler struct {
	svc inventory.Service
}

func NewHandler(svc inventory.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/inventory")

	// Item endpoints
	g.POST("/items", h.createItem)
	g.GET("/items", h.listItems)
	g.GET("/items/:id", h.getItem)
	g.PUT("/items/:id", h.updateItem)
	g.DELETE("/items/:id", h.deactivateItem)
	g.GET("/items/code/:code", h.getItemByCode)

	// Warehouse endpoints
	g.POST("/warehouses", h.createWarehouse)
	g.GET("/warehouses", h.listWarehouses)
	g.GET("/warehouses/:id", h.getWarehouse)
	g.PUT("/warehouses/:id", h.updateWarehouse)
	g.DELETE("/warehouses/:id", h.deactivateWarehouse)
	g.GET("/warehouses/code/:code", h.getWarehouseByCode)

	// Stock balance endpoints
	g.GET("/stock", h.listStockBalances)
	g.GET("/stock/:itemCode/:warehouseCode", h.getStockBalance)

	// Movement endpoints
	g.POST("/movements", h.createMovement)
	g.GET("/movements", h.listMovements)
	g.GET("/movements/:id", h.getMovement)
	g.POST("/movements/:id/confirm", h.confirmMovement)

	// Convenience endpoints
	g.POST("/transfer", h.transferStock)
	g.POST("/adjust", h.adjustStock)

	// Physical Count endpoints
	g.POST("/counts", h.createPhysicalCount)
	g.GET("/counts", h.listPhysicalCounts)
	g.GET("/counts/:id", h.getPhysicalCount)
	g.PUT("/counts/:id", h.updatePhysicalCount)
	g.POST("/counts/:id/complete", h.completePhysicalCount)
	g.POST("/counts/:id/reconcile", h.reconcilePhysicalCount)

	// NRV Write-Down endpoints
	g.POST("/writedown", h.writeDownNRV)
	g.POST("/writedown/reverse", h.reverseWriteDown)

	// Opening Balance endpoint
	g.POST("/opening-balance", h.importOpeningBalance)

	// Report endpoints
	g.GET("/reports/balance", h.getStockBalanceReport)
	g.GET("/reports/movements", h.getStockMovementReport)
	g.GET("/reports/valuation", h.getStockValuationReport)
}

func (h *Handler) actor(c *gin.Context) string {
	if u := c.GetHeader("X-User-Id"); u != "" {
		return u
	}
	return "system"
}

func respondError(c *gin.Context, err error) {
	var ve *core.ValidationError
	if errors.As(err, &ve) {
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message, "field": ve.Field})
		return
	}
	switch {
	case errors.Is(err, domaininventory.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domaininventory.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domaininventory.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// --- Item handlers ---

func (h *Handler) createItem(c *gin.Context) {
	var i domaininventory.Item
	if err := c.ShouldBindJSON(&i); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.svc.CreateItem(c.Request.Context(), &i, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) getItem(c *gin.Context) {
	i, err := h.svc.GetItem(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, i)
}

func (h *Handler) getItemByCode(c *gin.Context) {
	i, err := h.svc.GetItemByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, i)
}

func (h *Handler) updateItem(c *gin.Context) {
	var i domaininventory.Item
	if err := c.ShouldBindJSON(&i); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.svc.UpdateItem(c.Request.Context(), c.Param("id"), &i, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deactivateItem(c *gin.Context) {
	if err := h.svc.DeactivateItem(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listItems(c *gin.Context) {
	category := domaininventory.ItemCategory(c.Query("category"))
	status := domaininventory.ItemStatus(c.Query("status"))
	search := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, total, err := h.svc.ListItems(c.Request.Context(), category, status, search, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// --- Warehouse handlers ---

func (h *Handler) createWarehouse(c *gin.Context) {
	var w domaininventory.Warehouse
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.svc.CreateWarehouse(c.Request.Context(), &w, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) getWarehouse(c *gin.Context) {
	w, err := h.svc.GetWarehouse(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, w)
}

func (h *Handler) getWarehouseByCode(c *gin.Context) {
	w, err := h.svc.GetWarehouseByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, w)
}

func (h *Handler) updateWarehouse(c *gin.Context) {
	var w domaininventory.Warehouse
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.svc.UpdateWarehouse(c.Request.Context(), c.Param("id"), &w, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deactivateWarehouse(c *gin.Context) {
	if err := h.svc.DeactivateWarehouse(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listWarehouses(c *gin.Context) {
	status := domaininventory.WarehouseStatus(c.Query("status"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	warehouses, total, err := h.svc.ListWarehouses(c.Request.Context(), status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"warehouses": warehouses, "total": total})
}

// --- Stock balance handlers ---

func (h *Handler) getStockBalance(c *gin.Context) {
	sc, err := h.svc.GetStockBalance(c.Request.Context(), c.Param("itemCode"), c.Param("warehouseCode"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, sc)
}

func (h *Handler) listStockBalances(c *gin.Context) {
	warehouseCode := c.Query("warehouse")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	cards, total, err := h.svc.ListStockBalances(c.Request.Context(), warehouseCode, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"stock_cards": cards, "total": total})
}

// --- Movement handlers ---

func (h *Handler) createMovement(c *gin.Context) {
	var m domaininventory.StockMovement
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	movement, err := h.svc.CreateMovement(c.Request.Context(), &m, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, movement)
}

func (h *Handler) getMovement(c *gin.Context) {
	m, err := h.svc.GetMovement(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) confirmMovement(c *gin.Context) {
	m, err := h.svc.ConfirmMovement(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) listMovements(c *gin.Context) {
	itemCode := c.Query("item")
	warehouseCode := c.Query("warehouse")
	movementType := domaininventory.MovementType(c.Query("type"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	movements, total, err := h.svc.ListMovements(c.Request.Context(), itemCode, warehouseCode, movementType, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"movements": movements, "total": total})
}

// --- Convenience endpoints ---

func (h *Handler) transferStock(c *gin.Context) {
	var req struct {
		ItemCode      string  `json:"item_code"`
		FromWarehouse string  `json:"from_warehouse"`
		ToWarehouse   string  `json:"to_warehouse"`
		Quantity      float64 `json:"quantity"`
		UnitPrice     int64   `json:"unit_price"`
		MovementDate  string  `json:"movement_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, in, err := h.svc.TransferStock(c.Request.Context(), req.ItemCode, req.FromWarehouse, req.ToWarehouse, req.Quantity, req.UnitPrice, req.MovementDate, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"outbound": out, "inbound": in})
}

func (h *Handler) adjustStock(c *gin.Context) {
	var req struct {
		ItemCode      string  `json:"item_code"`
		WarehouseCode string  `json:"warehouse_code"`
		Quantity      float64 `json:"quantity"`
		UnitPrice     int64   `json:"unit_price"`
		Reason        string  `json:"reason"`
		MovementDate  string  `json:"movement_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mov, err := h.svc.AdjustStock(c.Request.Context(), req.ItemCode, req.WarehouseCode, req.Quantity, req.UnitPrice, req.Reason, req.MovementDate, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, mov)
}

// --- Physical Count handlers ---

func (h *Handler) createPhysicalCount(c *gin.Context) {
	var pc domaininventory.PhysicalCount
	if err := c.ShouldBindJSON(&pc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	count, err := h.svc.CreatePhysicalCount(c.Request.Context(), &pc, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, count)
}

func (h *Handler) getPhysicalCount(c *gin.Context) {
	pc, err := h.svc.GetPhysicalCount(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, pc)
}

func (h *Handler) updatePhysicalCount(c *gin.Context) {
	var patch domaininventory.PhysicalCount
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pc, err := h.svc.UpdatePhysicalCount(c.Request.Context(), c.Param("id"), &patch, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, pc)
}

func (h *Handler) completePhysicalCount(c *gin.Context) {
	pc, err := h.svc.CompletePhysicalCount(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, pc)
}

func (h *Handler) reconcilePhysicalCount(c *gin.Context) {
	movements, err := h.svc.ReconcilePhysicalCount(c.Request.Context(), c.Param("id"), h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"movements": movements, "count": len(movements)})
}

func (h *Handler) listPhysicalCounts(c *gin.Context) {
	warehouseCode := c.Query("warehouse")
	status := domaininventory.PhysicalCountStatus(c.Query("status"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	counts, total, err := h.svc.ListPhysicalCounts(c.Request.Context(), warehouseCode, status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"counts": counts, "total": total})
}

// --- NRV Write-Down handlers ---

func (h *Handler) writeDownNRV(c *gin.Context) {
	var req struct {
		ItemCode      string `json:"item_code"`
		WarehouseCode string `json:"warehouse_code"`
		NRVUnitCost   int64  `json:"nrv_unit_cost"`
		MovementDate  string `json:"movement_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mov, err := h.svc.WriteDownNRV(c.Request.Context(), req.ItemCode, req.WarehouseCode, req.NRVUnitCost, req.MovementDate, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, mov)
}

func (h *Handler) reverseWriteDown(c *gin.Context) {
	var req struct {
		ItemCode      string `json:"item_code"`
		WarehouseCode string `json:"warehouse_code"`
		NRVUnitCost   int64  `json:"nrv_unit_cost"`
		MovementDate  string `json:"movement_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mov, err := h.svc.ReverseWriteDown(c.Request.Context(), req.ItemCode, req.WarehouseCode, req.NRVUnitCost, req.MovementDate, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, mov)
}

// --- Report handlers ---

func (h *Handler) getStockBalanceReport(c *gin.Context) {
	warehouseCode := c.Query("warehouse_code")
	cards, _, err := h.svc.ListStockBalances(c.Request.Context(), warehouseCode, 1000, 0)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, cards)
}

func (h *Handler) getStockMovementReport(c *gin.Context) {
	itemCode := c.Query("item_code")
	warehouseCode := c.Query("warehouse_code")
	movements, _, err := h.svc.ListMovements(c.Request.Context(), itemCode, warehouseCode, "", 1000, 0)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, movements)
}

func (h *Handler) getStockValuationReport(c *gin.Context) {
	itemCode := c.Query("item_code")
	warehouseCode := c.Query("warehouse_code")
	cards, _, err := h.svc.ListStockBalances(c.Request.Context(), warehouseCode, 1000, 0)
	if err != nil {
		respondError(c, err)
		return
	}
	// Filter by itemCode if provided
	var result []*domaininventory.StockCard
	for _, card := range cards {
		if itemCode == "" || card.ItemCode == itemCode {
			result = append(result, card)
		}
	}
	c.JSON(http.StatusOK, result)
}

// --- Opening Balance handler ---

func (h *Handler) importOpeningBalance(c *gin.Context) {
	var req struct {
		ItemCode      string  `json:"item_code"`
		WarehouseCode string  `json:"warehouse_code"`
		Quantity      float64 `json:"quantity"`
		UnitPrice     int64   `json:"unit_price"`
		MovementDate  string  `json:"movement_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mov := &domaininventory.StockMovement{
		MovementType:  domaininventory.MovementOpeningBalance,
		MovementDate:  req.MovementDate,
		ItemCode:      req.ItemCode,
		WarehouseCode: req.WarehouseCode,
		Quantity:      req.Quantity,
		UnitPrice:     req.UnitPrice,
		TotalCost:     int64(req.Quantity) * req.UnitPrice,
	}

	mov2, err := h.svc.CreateMovement(c.Request.Context(), mov, h.actor(c))
	if err != nil {
		respondError(c, err)
		return
	}

	if _, err := h.svc.ConfirmMovement(c.Request.Context(), mov2.ID, h.actor(c)); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "opening balance imported"})
}
