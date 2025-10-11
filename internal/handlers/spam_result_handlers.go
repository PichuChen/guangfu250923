package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"guangfu250923/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type spamResultCreateInput struct {
	TargetID   string                 `json:"target_id" binding:"required"`
	TargetType string                 `json:"target_type" binding:"required"`
	TargetData map[string]interface{} `json:"target_data" binding:"required"`
	IsSpam     bool                   `json:"is_spam"`
	Judgment   string                 `json:"judgment" binding:"required"`
}

type spamResultPatchInput struct {
	IsSpam     *bool                   `json:"is_spam"`
	Judgment   *string                 `json:"judgment"`
	TargetData *map[string]interface{} `json:"target_data"`
}

func (h *Handler) CreateSpamResult(c *gin.Context) {
	var in spamResultCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newUUID, err := uuid.NewV7()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate id"})
		return
	}

	// Validate required fields
	for field, val := range map[string]string{"target_id": in.TargetID, "target_type": in.TargetType, "judgment": in.Judgment} {
		if strings.TrimSpace(val) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": field + " is required"})
			return
		}
	}
	validatedAt := time.Now().Unix()
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `insert into spam_result(id,target_id,target_type,target_data,is_spam,judgment,validated_at) values($1,$2,$3,$4,$5,$6,$7) returning id,target_id,target_type,target_data,is_spam,judgment,validated_at`,
		newUUID.String(), in.TargetID, in.TargetType, in.TargetData, in.IsSpam, in.Judgment, validatedAt)
	var sr models.SpamResult
	if err := row.Scan(&sr.ID, &sr.TargetID, &sr.TargetType, &sr.TargetData, &sr.IsSpam, &sr.Judgment, &sr.ValidatedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sr)
}

func (h *Handler) ListSpamResults(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	targetType := strings.TrimSpace(c.Query("target_type"))
	targetID := strings.TrimSpace(c.Query("target_id"))
	isSpamStr := strings.TrimSpace(c.Query("is_spam"))

	ctx := context.Background()
	filters := []string{}
	args := []interface{}{}

	if targetType != "" {
		filters = append(filters, "target_type=$"+strconv.Itoa(len(args)+1))
		args = append(args, targetType)
	}
	if targetID != "" {
		filters = append(filters, "target_id=$"+strconv.Itoa(len(args)+1))
		args = append(args, targetID)
	}
	if isSpamStr == "true" || isSpamStr == "false" {
		filters = append(filters, "is_spam=$"+strconv.Itoa(len(args)+1))
		args = append(args, isSpamStr == "true")
	}

	countSQL := `select count(*) from spam_result`
	listSQL := `select id,target_id,target_type,target_data,is_spam,judgment,validated_at from spam_result`
	if len(filters) > 0 {
		where := " where " + strings.Join(filters, " and ")
		countSQL += where
		listSQL += where
	}

	var total int
	if err := h.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	listSQL += " order by validated_at desc limit $" + strconv.Itoa(len(args)+1) + " offset $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := h.pool.Query(ctx, listSQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []models.SpamResult{}
	for rows.Next() {
		var sr models.SpamResult
		if err := rows.Scan(&sr.ID, &sr.TargetID, &sr.TargetType, &sr.TargetData, &sr.IsSpam, &sr.Judgment, &sr.ValidatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, sr)
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

func (h *Handler) GetSpamResult(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,target_id,target_type,target_data,is_spam,judgment,validated_at from spam_result where id=$1`, id)
	var sr models.SpamResult
	if err := row.Scan(&sr.ID, &sr.TargetID, &sr.TargetType, &sr.TargetData, &sr.IsSpam, &sr.Judgment, &sr.ValidatedAt); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sr)
}

func (h *Handler) PatchSpamResult(c *gin.Context) {
	id := c.Param("id")
	var in spamResultPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	setParts := []string{}
	args := []interface{}{}
	idx := 1
	add := func(expr string, val interface{}) {
		setParts = append(setParts, expr+"$"+strconv.Itoa(idx))
		args = append(args, val)
		idx++
	}
	if in.IsSpam != nil {
		add("is_spam=", *in.IsSpam)
	}
	if in.Judgment != nil {
		add("judgment=", *in.Judgment)
	}
	if in.TargetData != nil {
		add("target_data=", *in.TargetData)
	}
	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}
	query := "update spam_result set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,target_id,target_type,target_data,is_spam,judgment,validated_at"
	args = append(args, id)
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, query, args...)
	var sr models.SpamResult
	if err := row.Scan(&sr.ID, &sr.TargetID, &sr.TargetType, &sr.TargetData, &sr.IsSpam, &sr.Judgment, &sr.ValidatedAt); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sr)
}
