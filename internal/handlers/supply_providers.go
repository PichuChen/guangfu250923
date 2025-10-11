package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"guangfu250923/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type supplyProviderCreateInput struct {
	Name         string  `json:"name" binding:"required"`
	Phone        string  `json:"phone" binding:"required"`
	SupplyItemID string  `json:"supply_item_id" binding:"required"`
	Address      string  `json:"address" binding:"required"`
	Notes        *string `json:"notes"`
	ProvideCount int     `json:"provide_count" binding:"required"`
	ProvideUnit  *string `json:"provide_unit"`
}

func (h *Handler) CreateSupplyProvider(c *gin.Context) {
	var in supplyProviderCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	// Verify supply_item_id exists
	var exists bool
	if err := h.pool.QueryRow(ctx, `select exists(select 1 from supply_items where id=$1)`, in.SupplyItemID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found", "reason": "supply item not found"})
		return
	}

	newUUID, err := uuid.NewV7()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate id"})
		return
	}
	id := newUUID.String()

	var created, updated int64
	err = h.pool.QueryRow(ctx, `insert into supply_providers(id,name,phone,supply_item_id,address,notes,provide_count,provide_unit) values($1,$2,$3,$4,$5,$6,$7,$8) returning extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
		id, in.Name, in.Phone, in.SupplyItemID, in.Address, in.Notes, in.ProvideCount, in.ProvideUnit).Scan(&created, &updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.SupplyProvider{
		ID:           id,
		Name:         in.Name,
		Phone:        in.Phone,
		SupplyItemID: in.SupplyItemID,
		Address:      in.Address,
		Notes:        in.Notes,
		ProvideCount: in.ProvideCount,
		ProvideUnit:  in.ProvideUnit,
		CreatedAt:    created,
		UpdatedAt:    updated,
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) ListSupplyProviders(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	supplyItemID := c.Query("supply_item_id")
	ctx := context.Background()

	var total int
	var rows pgx.Rows
	var err error

	if supplyItemID != "" {
		if err := h.pool.QueryRow(ctx, `select count(*) from supply_providers where supply_item_id=$1`, supplyItemID).Scan(&total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rows, err = h.pool.Query(ctx, `select id,name,phone,supply_item_id,address,notes,provide_count,provide_unit,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from supply_providers where supply_item_id=$1 order by updated_at desc limit $2 offset $3`, supplyItemID, limit, offset)
	} else {
		if err := h.pool.QueryRow(ctx, `select count(*) from supply_providers`).Scan(&total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rows, err = h.pool.Query(ctx, `select id,name,phone,supply_item_id,address,notes,provide_count,provide_unit,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from supply_providers order by updated_at desc limit $1 offset $2`, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []models.SupplyProvider{}
	for rows.Next() {
		var sp models.SupplyProvider
		var created, updated int64
		if err = rows.Scan(&sp.ID, &sp.Name, &sp.Phone, &sp.SupplyItemID, &sp.Address, &sp.Notes, &sp.ProvideCount, &sp.ProvideUnit, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sp.CreatedAt = created
		sp.UpdatedAt = updated
		list = append(list, sp)
	}
	baseURL := c.Request.URL.Path
	q := c.Request.URL.Query()

	build := func(off int) string {
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(off))
		return baseURL + "?" + q.Encode()
	}
	var next *string
	if offset+limit < total {
		s := build(offset + limit)
		next = &s
	}
	var prev *string
	if offset-limit >= 0 {
		s := build(offset - limit)
		prev = &s
	}
	c.JSON(http.StatusOK, gin.H{"@context": "https://www.w3.org/ns/hydra/context.jsonld", "@type": "Collection", "totalItems": total, "member": list, "limit": limit, "offset": offset, "next": next, "previous": prev})
}

func (h *Handler) GetSupplyProvider(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,name,phone,supply_item_id,address,notes,provide_count,provide_unit,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from supply_providers where id=$1`, id)

	var sp models.SupplyProvider
	var created, updated int64
	if err := row.Scan(&sp.ID, &sp.Name, &sp.Phone, &sp.SupplyItemID, &sp.Address, &sp.Notes, &sp.ProvideCount, &sp.ProvideUnit, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sp.CreatedAt = created
	sp.UpdatedAt = updated
	c.JSON(http.StatusOK, sp)
}

type supplyProviderPatchInput struct {
	Name         *string `json:"name"`
	Phone        *string `json:"phone"`
	SupplyItemID *string `json:"supply_item_id"`
	Address      *string `json:"address"`
	Notes        *string `json:"notes"`
	ProvideCount *int    `json:"provide_count"`
	ProvideUnit  *string `json:"provide_unit"`
}

func (h *Handler) PatchSupplyProvider(c *gin.Context) {
	id := c.Param("id")
	var in supplyProviderPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	// If updating supply_item_id, verify it exists
	if in.SupplyItemID != nil {
		var exists bool
		if err := h.pool.QueryRow(ctx, `select exists(select 1 from supply_items where id=$1)`, *in.SupplyItemID).Scan(&exists); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found", "reason": "supply item not found"})
			return
		}
	}
	// Build dynamic update
	setParts := []string{}
	args := []interface{}{}
	idx := 1
	add := func(expr string, val interface{}) {
		setParts = append(setParts, expr+"$"+strconv.Itoa(idx))
		args = append(args, val)
		idx++
	}
	if in.Name != nil {
		add("name=", *in.Name)
	}
	if in.Phone != nil {
		add("phone=", *in.Phone)
	}
	if in.SupplyItemID != nil {
		add("supply_item_id=", *in.SupplyItemID)
	}
	if in.Address != nil {
		add("address=", *in.Address)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.ProvideCount != nil {
		add("provide_count=", *in.ProvideCount)
	}
	if in.ProvideUnit != nil {
		add("provide_unit=", *in.ProvideUnit)
	}
	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}
	// always update updated_at
	setParts = append(setParts, "updated_at=now()")
	query := "update supply_providers set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,name,phone,supply_item_id,address,notes,provide_count,provide_unit,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	row := h.pool.QueryRow(ctx, query, args...)
	var sp models.SupplyProvider
	var created, updated int64
	if err := row.Scan(&sp.ID, &sp.Name, &sp.Phone, &sp.SupplyItemID, &sp.Address, &sp.Notes, &sp.ProvideCount, &sp.ProvideUnit, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sp.CreatedAt = created
	sp.UpdatedAt = updated
	c.JSON(http.StatusOK, sp)
}
