package handlers

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"
    "strings"

    "guangfu250923/internal/models"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

type requirementsHRCreateInput struct {
    PlaceID      string   `json:"place_id" binding:"required"`
    RequiredType string   `json:"required_type" binding:"required"`
    Name         string   `json:"name" binding:"required"`
    Unit         string   `json:"unit" binding:"required"`
    RequireCount int      `json:"require_count" binding:"required"`
    ReceivedCount int     `json:"received_count"`
    Tags         []map[string]interface{} `json:"tags"`
    AdditionalInfo map[string]interface{} `json:"additional_info"`
}

func (h *Handler) CreateRequirementsHR(c *gin.Context) {
    var in requirementsHRCreateInput
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // Optional: verify place exists
    var exists bool
    if err := h.pool.QueryRow(context.Background(), `select exists(select 1 from places where id=$1)`, in.PlaceID).Scan(&exists); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }
    if !exists {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found", "reason": "place not found"}); return
    }
    var tagsJSON, addInfoJSON *string
    if in.Tags != nil { if b, err := json.Marshal(in.Tags); err == nil { s := string(b); tagsJSON = &s } }
    if in.AdditionalInfo != nil { if b, err := json.Marshal(in.AdditionalInfo); err == nil { s := string(b); addInfoJSON = &s } }
    newID, _ := uuid.NewV7()
    id := newID.String()
    var created, updated int64
    err := h.pool.QueryRow(context.Background(), `insert into requirements_hr(
        id,place_id,required_type,name,unit,require_count,received_count,tags,additional_info
    ) values($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb) returning extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
        id, in.PlaceID, in.RequiredType, in.Name, in.Unit, in.RequireCount, in.ReceivedCount, tagsJSON, addInfoJSON,
    ).Scan(&created, &updated)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    out := models.RequirementsHR{ID: id, PlaceID: in.PlaceID, RequiredType: in.RequiredType, Name: in.Name, Unit: in.Unit, RequireCount: in.RequireCount, ReceivedCount: in.ReceivedCount, CreatedAt: created, UpdatedAt: updated}
    out.Tags = in.Tags; out.AdditionalInfo = in.AdditionalInfo
    c.JSON(http.StatusCreated, out)
}

func (h *Handler) GetRequirementsHR(c *gin.Context) {
    id := c.Param("id")
    row := h.pool.QueryRow(context.Background(), `select id,place_id,required_type,name,unit,require_count,received_count,tags,additional_info,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from requirements_hr where id=$1`, id)
    var r models.RequirementsHR
    var tagsJSON, addInfoJSON []byte
    if err := row.Scan(&r.ID, &r.PlaceID, &r.RequiredType, &r.Name, &r.Unit, &r.RequireCount, &r.ReceivedCount, &tagsJSON, &addInfoJSON, &r.CreatedAt, &r.UpdatedAt); err != nil {
        if err == pgx.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }
    if len(tagsJSON) > 0 { var arr []map[string]interface{}; _ = json.Unmarshal(tagsJSON, &arr); r.Tags = arr }
    if len(addInfoJSON) > 0 { var m map[string]interface{}; _ = json.Unmarshal(addInfoJSON, &m); r.AdditionalInfo = m }
    c.JSON(http.StatusOK, r)
}

func (h *Handler) ListRequirementsHR(c *gin.Context) {
    limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
    offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
    placeID := c.Query("place_id")
    reqType := c.Query("required_type")
    filters := []string{}
    args := []interface{}{}
    if placeID != "" { filters = append(filters, "place_id=$"+strconv.Itoa(len(args)+1)); args = append(args, placeID) }
    if reqType != "" { filters = append(filters, "required_type=$"+strconv.Itoa(len(args)+1)); args = append(args, reqType) }
    countQ := "select count(*) from requirements_hr"
    dataQ := "select id,place_id,required_type,name,unit,require_count,received_count,tags,additional_info,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from requirements_hr"
    if len(filters) > 0 { where := " where "+strings.Join(filters, " and "); countQ += where; dataQ += where }
    var total int
    if err := h.pool.QueryRow(context.Background(), countQ, args...).Scan(&total); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    args = append(args, limit, offset)
    dataQ += " order by updated_at desc limit $"+strconv.Itoa(len(args)-1)+" offset $"+strconv.Itoa(len(args))
    rows, err := h.pool.Query(context.Background(), dataQ, args...)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    defer rows.Close()
    list := []models.RequirementsHR{}
    for rows.Next() {
        var r models.RequirementsHR
        var tagsJSON, addInfoJSON []byte
        if err := rows.Scan(&r.ID, &r.PlaceID, &r.RequiredType, &r.Name, &r.Unit, &r.RequireCount, &r.ReceivedCount, &tagsJSON, &addInfoJSON, &r.CreatedAt, &r.UpdatedAt); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
        }
        if len(tagsJSON) > 0 { var arr []map[string]interface{}; _ = json.Unmarshal(tagsJSON, &arr); r.Tags = arr }
        if len(addInfoJSON) > 0 { var m map[string]interface{}; _ = json.Unmarshal(addInfoJSON, &m); r.AdditionalInfo = m }
        list = append(list, r)
    }
    baseURL := c.Request.URL.Path
    q := c.Request.URL.Query()
    build := func(off int) string { q.Set("limit", strconv.Itoa(limit)); q.Set("offset", strconv.Itoa(off)); return baseURL+"?"+q.Encode() }
    var next *string
    if offset+limit < total { s := build(offset+limit); next = &s }
    var prev *string
    if offset-limit >= 0 { s := build(offset-limit); prev = &s }
    c.JSON(http.StatusOK, gin.H{"@context": "https://www.w3.org/ns/hydra/context.jsonld", "@type": "Collection", "totalItems": total, "member": list, "limit": limit, "offset": offset, "next": next, "previous": prev})
}

type requirementsHRPatchInput struct {
    PlaceID       *string `json:"place_id"`
    RequiredType  *string `json:"required_type"`
    Name          *string `json:"name"`
    Unit          *string `json:"unit"`
    RequireCount  *int    `json:"require_count"`
    ReceivedCount *int    `json:"received_count"`
    Tags          *[]map[string]interface{} `json:"tags"`
    AdditionalInfo *map[string]interface{}  `json:"additional_info"`
}

func (h *Handler) PatchRequirementsHR(c *gin.Context) {
    id := c.Param("id")
    var in requirementsHRPatchInput
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    setParts := []string{}
    args := []interface{}{}
    idx := 1
    add := func(expr string, val interface{}) { setParts = append(setParts, expr+"$"+strconv.Itoa(idx)); args = append(args, val); idx++ }
    if in.PlaceID != nil { add("place_id=", *in.PlaceID) }
    if in.RequiredType != nil { add("required_type=", *in.RequiredType) }
    if in.Name != nil { add("name=", *in.Name) }
    if in.Unit != nil { add("unit=", *in.Unit) }
    if in.RequireCount != nil { add("require_count=", *in.RequireCount) }
    if in.ReceivedCount != nil { add("received_count=", *in.ReceivedCount) }
    if in.Tags != nil { if b, err := json.Marshal(in.Tags); err == nil { setParts = append(setParts, "tags=$"+strconv.Itoa(idx)+"::jsonb"); args = append(args, string(b)); idx++ } }
    if in.AdditionalInfo != nil { if b, err := json.Marshal(in.AdditionalInfo); err == nil { setParts = append(setParts, "additional_info=$"+strconv.Itoa(idx)+"::jsonb"); args = append(args, string(b)); idx++ } }
    if len(setParts) == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"}); return }
    setParts = append(setParts, "updated_at=now()")
    query := "update requirements_hr set "+strings.Join(setParts, ",")+" where id=$"+strconv.Itoa(idx)+" returning id,place_id,required_type,name,unit,require_count,received_count,tags,additional_info,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
    args = append(args, id)
    row := h.pool.QueryRow(context.Background(), query, args...)
    var r models.RequirementsHR
    var tagsJSON, addInfoJSON []byte
    if err := row.Scan(&r.ID, &r.PlaceID, &r.RequiredType, &r.Name, &r.Unit, &r.RequireCount, &r.ReceivedCount, &tagsJSON, &addInfoJSON, &r.CreatedAt, &r.UpdatedAt); err != nil {
        if err == pgx.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }
    if len(tagsJSON) > 0 { var arr []map[string]interface{}; _ = json.Unmarshal(tagsJSON, &arr); r.Tags = arr }
    if len(addInfoJSON) > 0 { var m map[string]interface{}; _ = json.Unmarshal(addInfoJSON, &m); r.AdditionalInfo = m }
    c.JSON(http.StatusOK, r)
}
