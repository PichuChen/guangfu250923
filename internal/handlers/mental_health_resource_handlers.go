package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"guangfu250923/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type mentalHealthResourceCreateInput struct {
	DurationType   string   `json:"duration_type" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	ServiceFormat  string   `json:"service_format" binding:"required"`
	ServiceHours   string   `json:"service_hours" binding:"required"`
	ContactInfo    string   `json:"contact_info" binding:"required"`
	WebsiteURL     *string  `json:"website_url"`
	TargetAudience []string `json:"target_audience"`
	Specialties    []string `json:"specialties"`
	Languages      []string `json:"languages"`
	IsFree         *bool    `json:"is_free" binding:"required"`
	Location       *string  `json:"location"`
	Coordinates    *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	Status           string  `json:"status" binding:"required"`
	Capacity         *int    `json:"capacity"`
	WaitingTime      *string `json:"waiting_time"`
	Notes            *string `json:"notes"`
	EmergencySupport *bool   `json:"emergency_support" binding:"required"`
}

func (h *Handler) CreateMentalHealthResource(c *gin.Context) {
	var in mentalHealthResourceCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	isFree := false
	if in.IsFree != nil {
		isFree = *in.IsFree
	}
	emergency := false
	if in.EmergencySupport != nil {
		emergency = *in.EmergencySupport
	}
	var coordsJSON *string
	if in.Coordinates != nil {
		if b, err := json.Marshal(in.Coordinates); err == nil {
			s := string(b)
			coordsJSON = &s
		}
	}
	var id string
	var created, updated int64
	err := h.pool.QueryRow(ctx, `insert into mental_health_resources(duration_type,name,service_format,service_hours,contact_info,website_url,target_audience,specialties,languages,is_free,location,coordinates,status,capacity,waiting_time,notes,emergency_support) values($1,$2,$3,$4,$5,$6,$7::text[],$8::text[],$9::text[],$10,$11,$12::jsonb,$13,$14,$15,$16,$17) returning id,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
		in.DurationType, in.Name, in.ServiceFormat, in.ServiceHours, in.ContactInfo, in.WebsiteURL, in.TargetAudience, in.Specialties, in.Languages, isFree, in.Location, coordsJSON, in.Status, in.Capacity, in.WaitingTime, in.Notes, emergency).Scan(&id, &created, &updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.MentalHealthResource{ID: id, DurationType: in.DurationType, Name: in.Name, ServiceFormat: in.ServiceFormat, ServiceHours: in.ServiceHours, ContactInfo: in.ContactInfo, WebsiteURL: in.WebsiteURL, TargetAudience: in.TargetAudience, Specialties: in.Specialties, Languages: in.Languages, IsFree: isFree, Location: in.Location, Status: in.Status, Capacity: in.Capacity, WaitingTime: in.WaitingTime, Notes: in.Notes, EmergencySupport: emergency, CreatedAt: created, UpdatedAt: updated}
	out.Coordinates = in.Coordinates
	c.JSON(http.StatusCreated, out)
}

type mentalHealthResourcePatchInput struct {
	DurationType   *string   `json:"duration_type"`
	Name           *string   `json:"name"`
	ServiceFormat  *string   `json:"service_format"`
	ServiceHours   *string   `json:"service_hours"`
	ContactInfo    *string   `json:"contact_info"`
	WebsiteURL     *string   `json:"website_url"`
	TargetAudience *[]string `json:"target_audience"`
	Specialties    *[]string `json:"specialties"`
	Languages      *[]string `json:"languages"`
	IsFree         *bool     `json:"is_free"`
	Location       *string   `json:"location"`
	Coordinates    *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	Status           *string `json:"status"`
	Capacity         *int    `json:"capacity"`
	WaitingTime      *string `json:"waiting_time"`
	Notes            *string `json:"notes"`
	EmergencySupport *bool   `json:"emergency_support"`
}

func (h *Handler) PatchMentalHealthResource(c *gin.Context) {
	id := c.Param("id")
	var in mentalHealthResourcePatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	setParts := []string{}
	args := []interface{}{}
	idx := 1
	add := func(expr string, val interface{}) {
		setParts = append(setParts, expr+"$"+strconv.Itoa(idx))
		args = append(args, val)
		idx++
	}
	if in.DurationType != nil {
		add("duration_type=", *in.DurationType)
	}
	if in.Name != nil {
		add("name=", *in.Name)
	}
	if in.ServiceFormat != nil {
		add("service_format=", *in.ServiceFormat)
	}
	if in.ServiceHours != nil {
		add("service_hours=", *in.ServiceHours)
	}
	if in.ContactInfo != nil {
		add("contact_info=", *in.ContactInfo)
	}
	if in.WebsiteURL != nil {
		add("website_url=", *in.WebsiteURL)
	}
	if in.TargetAudience != nil {
		add("target_audience=", *in.TargetAudience)
	}
	if in.Specialties != nil {
		add("specialties=", *in.Specialties)
	}
	if in.Languages != nil {
		add("languages=", *in.Languages)
	}
	if in.IsFree != nil {
		add("is_free=", *in.IsFree)
	}
	if in.Location != nil {
		add("location=", *in.Location)
	}
	if in.Status != nil {
		add("status=", *in.Status)
	}
	if in.Capacity != nil {
		add("capacity=", *in.Capacity)
	}
	if in.WaitingTime != nil {
		add("waiting_time=", *in.WaitingTime)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.EmergencySupport != nil {
		add("emergency_support=", *in.EmergencySupport)
	}
	if in.Coordinates != nil {
		if b, err := json.Marshal(in.Coordinates); err == nil {
			setParts = append(setParts, "coordinates=$"+strconv.Itoa(idx)+"::jsonb")
			args = append(args, string(b))
			idx++
		}
	}
	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}
	setParts = append(setParts, "updated_at=now()")
	query := "update mental_health_resources set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,duration_type,name,service_format,service_hours,contact_info,website_url,target_audience,specialties,languages,is_free,location,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,status,capacity,waiting_time,notes,emergency_support,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	row := h.pool.QueryRow(ctx, query, args...)
	var m models.MentalHealthResource
	var websiteURL, location, waitingTime, notes *string
	var lat, lng *float64
	var capacity *int
	var targetAudience, specialties, languages []string
	var created, updated int64
	if err := row.Scan(&m.ID, &m.DurationType, &m.Name, &m.ServiceFormat, &m.ServiceHours, &m.ContactInfo, &websiteURL, &targetAudience, &specialties, &languages, &m.IsFree, &location, &lat, &lng, &m.Status, &capacity, &waitingTime, &notes, &m.EmergencySupport, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	m.WebsiteURL = websiteURL
	m.Location = location
	m.WaitingTime = waitingTime
	m.Notes = notes
	m.Capacity = capacity
	m.TargetAudience = targetAudience
	m.Specialties = specialties
	m.Languages = languages
	m.CreatedAt = created
	m.UpdatedAt = updated
	if lat != nil || lng != nil {
		m.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) GetMentalHealthResource(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,duration_type,name,service_format,service_hours,contact_info,website_url,target_audience,specialties,languages,is_free,location,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,status,capacity,waiting_time,notes,emergency_support,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from mental_health_resources where id=$1`, id)
	var m models.MentalHealthResource
	var websiteURL, location, waitingTime, notes *string
	var lat, lng *float64
	var capacity *int
	var targetAudience, specialties, languages []string
	var created, updated int64
	if err := row.Scan(&m.ID, &m.DurationType, &m.Name, &m.ServiceFormat, &m.ServiceHours, &m.ContactInfo, &websiteURL, &targetAudience, &specialties, &languages, &m.IsFree, &location, &lat, &lng, &m.Status, &capacity, &waitingTime, &notes, &m.EmergencySupport, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	m.WebsiteURL = websiteURL
	m.Location = location
	m.WaitingTime = waitingTime
	m.Notes = notes
	m.Capacity = capacity
	m.TargetAudience = targetAudience
	m.Specialties = specialties
	m.Languages = languages
	m.CreatedAt = created
	m.UpdatedAt = updated
	if lat != nil || lng != nil {
		m.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) ListMentalHealthResources(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	status := c.Query("status")
	duration := c.Query("duration_type")
	serviceFormat := c.Query("service_format")
	ctx := context.Background()
	filters := []string{}
	args := []interface{}{}
	if status != "" {
		filters = append(filters, "status=$"+strconv.Itoa(len(args)+1))
		args = append(args, status)
	}
	if duration != "" {
		filters = append(filters, "duration_type=$"+strconv.Itoa(len(args)+1))
		args = append(args, duration)
	}
	if serviceFormat != "" {
		filters = append(filters, "service_format=$"+strconv.Itoa(len(args)+1))
		args = append(args, serviceFormat)
	}
	countQ := "select count(*) from mental_health_resources"
	dataQ := "select id,duration_type,name,service_format,service_hours,contact_info,website_url,target_audience,specialties,languages,is_free,location,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,status,capacity,waiting_time,notes,emergency_support,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from mental_health_resources"
	if len(filters) > 0 {
		where := " where " + strings.Join(filters, " and ")
		countQ += where
		dataQ += where
	}
	var total int
	if err := h.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	args = append(args, limit, offset)
	dataQ += " order by updated_at desc limit $" + strconv.Itoa(len(args)-1) + " offset $" + strconv.Itoa(len(args))
	rows, err := h.pool.Query(ctx, dataQ, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []models.MentalHealthResource{}
	for rows.Next() {
		var m models.MentalHealthResource
		var websiteURL, location, waitingTime, notes *string
		var lat, lng *float64
		var capacity *int
		var targetAudience, specialties, languages []string
		var created, updated int64
		if err := rows.Scan(&m.ID, &m.DurationType, &m.Name, &m.ServiceFormat, &m.ServiceHours, &m.ContactInfo, &websiteURL, &targetAudience, &specialties, &languages, &m.IsFree, &location, &lat, &lng, &m.Status, &capacity, &waitingTime, &notes, &m.EmergencySupport, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		m.WebsiteURL = websiteURL
		m.Location = location
		m.WaitingTime = waitingTime
		m.Notes = notes
		m.Capacity = capacity
		m.TargetAudience = targetAudience
		m.Specialties = specialties
		m.Languages = languages
		m.CreatedAt = created
		m.UpdatedAt = updated
		if lat != nil || lng != nil {
			m.Coordinates = &struct {
				Lat *float64 `json:"lat"`
				Lng *float64 `json:"lng"`
			}{Lat: lat, Lng: lng}
		}
		list = append(list, m)
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
