package handlers

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	"guangfu250923/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type supplyCreateInput struct {
	Name     *string           `json:"name"`
	Address  *string           `json:"address"`
	Phone    *string           `json:"phone"`
	Notes    *string           `json:"notes"`
	PiiDate  *int64            `json:"pii_date"`
	Supplies *supplyItemInline `json:"supplies"`
	ValidPin *string           `json:"valid_pin"`
}

// Inline single item (前端需求: POST /supplies 時直接附上一個 supplies 物資項目)
type supplyItemInline struct {
	Tag           *string `json:"tag"`
	Name          *string `json:"name"`
	ReceivedCount *int    `json:"recieved_count"` // 注意: 前端拼字 recieved_count
	TotalCount    int     `json:"total_count" binding:"required"`
	Unit          *string `json:"unit"`
}

type supplyItemCreateInput struct { // 保留原獨立建立 endpoint 使用
	SupplyID   string  `json:"supply_id" binding:"required"`
	Tag        *string `json:"tag"`
	Name       *string `json:"name"`
	TotalCount int     `json:"total_count" binding:"required"`
	Unit       *string `json:"unit"`
}

func (h *Handler) CreateSupply(c *gin.Context) {
	var in supplyCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// PIN: generate if empty, else validate
	if in.ValidPin == nil || strings.TrimSpace(*in.ValidPin) == "" {
		tmp := GeneratePin(6)
		in.ValidPin = &tmp
	} else if !isValidPin6(in.ValidPin) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid_pin must be 6 digits"})
		return
	}
	ctx := context.Background()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)
	var id string
	var created, updated int64
	if err := tx.QueryRow(ctx, `insert into supplies(name,address,phone,notes,pii_date,valid_pin) values($1,$2,$3,$4,$5,$6) returning id,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`, in.Name, in.Address, in.Phone, in.Notes, in.PiiDate, in.ValidPin).Scan(&id, &created, &updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var createdItems []models.SupplyItem
	if in.Supplies != nil {
		received := 0
		if in.Supplies.ReceivedCount != nil {
			received = *in.Supplies.ReceivedCount
		}
		if received > in.Supplies.TotalCount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "recieved_count cannot exceed total_count"})
			return
		}
		var itemID string
		if err := tx.QueryRow(ctx, `insert into supply_items(supply_id,tag,name,received_count,total_number,unit) values($1,$2,$3,$4,$5,$6) returning id`, id, in.Supplies.Tag, in.Supplies.Name, received, in.Supplies.TotalCount, in.Supplies.Unit).Scan(&itemID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		createdItems = append(createdItems, models.SupplyItem{ID: itemID, SupplyID: id, Tag: in.Supplies.Tag, Name: in.Supplies.Name, ReceivedCount: received, TotalCount: in.Supplies.TotalCount, Unit: in.Supplies.Unit})
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := gin.H{"@context": "https://www.w3.org/ns/hydra/context.jsonld", "@type": "Supply", "id": id, "name": in.Name, "address": in.Address, "phone": in.Phone, "notes": in.Notes, "pii_date": in.PiiDate, "created_at": created, "updated_at": updated, "supplies": createdItems}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) ListSupplies(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	embed := c.Query("embed")
	ctx := context.Background()
	var total int
	if err := h.pool.QueryRow(ctx, `select count(*) from supplies`).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.pool.Query(ctx, `select id,name,address,phone,notes,pii_date,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from supplies order by updated_at desc limit $1 offset $2`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []models.Supply{}
	for rows.Next() {
		var s models.Supply
		var name, addr, phone, notes *string
		var piiDate *int64
		var created, updated int64
		if err := rows.Scan(&s.ID, &name, &addr, &phone, &notes, &piiDate, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.Name = name
		s.Address = addr
		s.Phone = phone
		s.Notes = notes
		s.PiiDate = piiDate
		s.CreatedAt = created
		s.UpdatedAt = updated
		list = append(list, s)
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
	// If embed=all, batch load all items; else keep empty arrays for consistency
	itemsMap := map[string][]models.SupplyItem{}
	if embed == "all" && len(list) > 0 {
		// Build IN clause dynamically
		placeholders := make([]string, len(list))
		argsItems := make([]interface{}, len(list))
		for i, s := range list {
			placeholders[i] = "$" + strconv.Itoa(i+1)
			argsItems[i] = s.ID
		}
		query := "select id,supply_id,tag,name,received_count,total_number,unit from supply_items where supply_id in (" + strings.Join(placeholders, ",") + ") order by supply_id,id asc"
		rowsIt, err := h.pool.Query(ctx, query, argsItems...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for rowsIt.Next() {
			var it models.SupplyItem
			var tag, name, unit *string
			if err := rowsIt.Scan(&it.ID, &it.SupplyID, &tag, &name, &it.ReceivedCount, &it.TotalCount, &unit); err != nil {
				rowsIt.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			it.Tag = tag
			it.Name = name
			it.Unit = unit
			itemsMap[it.SupplyID] = append(itemsMap[it.SupplyID], it)
		}
		rowsIt.Close()
	}
	wrapped := make([]gin.H, 0, len(list))
	for _, s := range list {
		var suppliesArr any = []interface{}{}
		if embed == "all" {
			if its, ok := itemsMap[s.ID]; ok {
				suppliesArr = its
			} else {
				suppliesArr = []interface{}{}
			}
		}
		wrapped = append(wrapped, gin.H{
			"id":         s.ID,
			"name":       s.Name,
			"address":    s.Address,
			"phone":      s.Phone,
			"notes":      s.Notes,
			"pii_date":   s.PiiDate,
			"created_at": s.CreatedAt,
			"updated_at": s.UpdatedAt,
			"supplies":   suppliesArr,
		})
	}
	c.JSON(http.StatusOK, gin.H{"@context": "https://www.w3.org/ns/hydra/context.jsonld", "@type": "Collection", "totalItems": total, "member": wrapped, "limit": limit, "offset": offset, "next": next, "previous": prev})
}

func (h *Handler) GetSupply(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,name,address,phone,notes,pii_date,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from supplies where id=$1`, id)
	var s models.Supply
	var name, addr, phone, notes *string
	var piiDate *int64
	var created, updated int64
	if err := row.Scan(&s.ID, &name, &addr, &phone, &notes, &piiDate, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.Name = name
	s.Address = addr
	s.Phone = phone
	s.Notes = notes
	s.PiiDate = piiDate
	s.CreatedAt = created
	s.UpdatedAt = updated
	// fetch ALL items (could be zero)
	rows, err := h.pool.Query(ctx, `select id,supply_id,tag,name,received_count,total_number,unit from supply_items where supply_id=$1 order by id asc`, s.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []models.SupplyItem{}
	for rows.Next() {
		var it models.SupplyItem
		var tag, iname, unit *string
		if err := rows.Scan(&it.ID, &it.SupplyID, &tag, &iname, &it.ReceivedCount, &it.TotalCount, &unit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		it.Tag = tag
		it.Name = iname
		it.Unit = unit
		items = append(items, it)
	}
	resp := gin.H{"@context": "https://www.w3.org/ns/hydra/context.jsonld", "@type": "Supply", "id": s.ID, "name": s.Name, "address": s.Address, "phone": s.Phone, "notes": s.Notes, "pii_date": s.PiiDate, "created_at": s.CreatedAt, "updated_at": s.UpdatedAt, "supplies": items}
	c.JSON(http.StatusOK, resp)
}

type supplyPatchInput struct {
	Name     *string `json:"name"`
	Address  *string `json:"address"`
	Phone    *string `json:"phone"`
	Notes    *string `json:"notes"`
	PiiDate  *int64  `json:"pii_date"`
	ValidPin *string `json:"valid_pin"`
}

func (h *Handler) PatchSupply(c *gin.Context) {
	id := c.Param("id")
	var in supplyPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Optional verification (controlled by VERIFY_SUPPLY_PIN)
	if os.Getenv("VERIFY_SUPPLY_PIN") == "true" {
		var storedPin *string
		if err := h.pool.QueryRow(context.Background(), `select valid_pin from supplies where id=$1`, id).Scan(&storedPin); err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if storedPin == nil || strings.TrimSpace(*storedPin) == "" {
			// bypass
		} else {
			if !isValidPin6(in.ValidPin) || *in.ValidPin != *storedPin {
				c.JSON(http.StatusForbidden, gin.H{"error": "invalid pin"})
				return
			}

		}
	}
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
	if in.Address != nil {
		add("address=", *in.Address)
	}
	if in.Phone != nil {
		add("phone=", *in.Phone)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.PiiDate != nil {
		add("pii_date=", *in.PiiDate)
	}
	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}
	setParts = append(setParts, "updated_at=now()")
	query := "update supplies set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,name,address,phone,notes,pii_date,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, query, args...)
	var s models.Supply
	var name, addr, phone, notes *string
	var piiDate *int64
	var created, updated int64
	if err := row.Scan(&s.ID, &name, &addr, &phone, &notes, &piiDate, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.Name = name
	s.Address = addr
	s.Phone = phone
	s.Notes = notes
	s.PiiDate = piiDate
	s.CreatedAt = created
	s.UpdatedAt = updated
	c.JSON(http.StatusOK, s)
}

func (h *Handler) CreateSupplyItem(c *gin.Context) {
	var in supplyItemCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	var id string
	err := h.pool.QueryRow(ctx, `insert into supply_items(supply_id,tag,name,total_number,unit) values($1,$2,$3,$4,$5) returning id`, in.SupplyID, in.Tag, in.Name, in.TotalCount, in.Unit).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) ListSupplyItems(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 100, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	supplyID := c.Query("supply_id")
	ctx := context.Background()
	filters := []string{}
	args := []interface{}{}
	if supplyID != "" {
		filters = append(filters, "supply_id=$"+strconv.Itoa(len(args)+1))
		args = append(args, supplyID)
	}
	countQuery := "select count(*) from supply_items"
	dataQuery := "select id,supply_id,tag,name,received_count,total_number,unit from supply_items"
	if len(filters) > 0 {
		where := " where " + strings.Join(filters, " and ")
		countQuery += where
		dataQuery += where
	}
	var total int
	if err := h.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	args = append(args, limit, offset)
	dataQuery += " order by id desc limit $" + strconv.Itoa(len(args)-1) + " offset $" + strconv.Itoa(len(args))
	rows, err := h.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []models.SupplyItem{}
	for rows.Next() {
		var it models.SupplyItem
		var tag, name, unit *string
		if err := rows.Scan(&it.ID, &it.SupplyID, &tag, &name, &it.ReceivedCount, &it.TotalCount, &unit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		it.Tag = tag
		it.Name = name
		it.Unit = unit
		list = append(list, it)
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

type supplyItemPatchInput struct {
	Tag           *string `json:"tag"`
	Name          *string `json:"name"`
	ReceivedCount *int    `json:"recieved_count"`
	TotalNumber   *int    `json:"total_count"`
	Unit          *string `json:"unit"`
}

func (h *Handler) PatchSupplyItem(c *gin.Context) {
	id := c.Param("id")
	var in supplyItemPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validation if counts involved
	if in.ReceivedCount != nil || in.TotalNumber != nil {
		ctxCheck := context.Background()
		var existingReceived, existingTotal int
		if err := h.pool.QueryRow(ctxCheck, "select received_count,total_number from supply_items where id=$1", id).Scan(&existingReceived, &existingTotal); err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		newReceived := existingReceived
		newTotal := existingTotal
		if in.ReceivedCount != nil {
			newReceived = *in.ReceivedCount
		}
		if in.TotalNumber != nil {
			newTotal = *in.TotalNumber
		}
		if newReceived > newTotal {
			c.JSON(http.StatusBadRequest, gin.H{"error": "recieved_count cannot exceed total_count"})
			return
		}
	}
	setParts := []string{}
	args := []interface{}{}
	idx := 1
	add := func(expr string, val interface{}) {
		setParts = append(setParts, expr+"$"+strconv.Itoa(idx))
		args = append(args, val)
		idx++
	}
	if in.Tag != nil {
		add("tag=", *in.Tag)
	}
	if in.Name != nil {
		add("name=", *in.Name)
	}
	if in.ReceivedCount != nil {
		add("received_count=", *in.ReceivedCount)
	}
	if in.TotalNumber != nil {
		add("total_number=", *in.TotalNumber)
	}
	if in.Unit != nil {
		add("unit=", *in.Unit)
	}
	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}
	query := "update supply_items set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,supply_id,tag,name,received_count,total_number,unit"
	args = append(args, id)
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, query, args...)
	var it models.SupplyItem
	var tag, name, unit *string
	if err := row.Scan(&it.ID, &it.SupplyID, &tag, &name, &it.ReceivedCount, &it.TotalCount, &unit); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	it.Tag = tag
	it.Name = name
	it.Unit = unit
	c.JSON(http.StatusOK, it)
}

func (h *Handler) GetSupplyItem(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,supply_id,tag,name,received_count,total_number,unit from supply_items where id=$1`, id)
	var it models.SupplyItem
	var tag, name, unit *string
	if err := row.Scan(&it.ID, &it.SupplyID, &tag, &name, &it.ReceivedCount, &it.TotalCount, &unit); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	it.Tag = tag
	it.Name = name
	it.Unit = unit
	c.JSON(http.StatusOK, it)
}

// POST /supplies/:id  (批次配送某供應單的多個物資項目)
type distributeItemInput struct {
	ID    string `json:"id" binding:"required"`
	Count int    `json:"count" binding:"required"`
}

func (h *Handler) DistributeSupplyItems(c *gin.Context) {
	supplyID := c.Param("id")
	var in []distributeItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(in) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty payload"})
		return
	}
	if len(in) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many items (max 500)"})
		return
	}
	ctx := context.Background()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)
	updated := []models.SupplyItem{}
	for _, itm := range in {
		if itm.Count <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "count must be > 0", "id": itm.ID})
			return
		}
		var curSuppID string
		var received, total int
		// lock row
		if err := tx.QueryRow(ctx, `select supply_id,received_count,total_number from supply_items where id=$1 for update`, itm.ID).Scan(&curSuppID, &received, &total); err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "item not found", "id": itm.ID})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "id": itm.ID})
			return
		}
		if curSuppID != supplyID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item does not belong to supply", "id": itm.ID})
			return
		}
		newReceived := received + itm.Count
		if newReceived > total {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exceeds total_count", "id": itm.ID, "recieved_count": received, "total_count": total, "attempt_add": itm.Count})
			return
		}
		var out models.SupplyItem
		var tag, name, unit *string
		if err := tx.QueryRow(ctx, `update supply_items set received_count=$1 where id=$2 returning id,supply_id,tag,name,received_count,total_number,unit`, newReceived, itm.ID).Scan(&out.ID, &out.SupplyID, &tag, &name, &out.ReceivedCount, &out.TotalCount, &unit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "id": itm.ID})
			return
		}
		out.Tag = tag
		out.Name = name
		out.Unit = unit
		updated = append(updated, out)
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}
