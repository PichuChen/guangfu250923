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

type waterRefillStationCreateInput struct {
	Name                   string   `json:"name" binding:"required"`
	Address                string   `json:"address" binding:"required"`
	Phone                  *string  `json:"phone"`
	WaterType              string   `json:"water_type" binding:"required"`
	OpeningHours           string   `json:"opening_hours" binding:"required"`
	IsFree                 *bool    `json:"is_free" binding:"required"`
	ContainerRequired      *string  `json:"container_required"`
	DailyCapacity          *int     `json:"daily_capacity"`
	Status                 string   `json:"status" binding:"required"`
	WaterQuality           *string  `json:"water_quality"`
	Facilities             []string `json:"facilities"`
	Accessibility          *bool    `json:"accessibility" binding:"required"`
	DistanceToDisasterArea *string  `json:"distance_to_disaster_area"`
	Notes                  *string  `json:"notes"`
	InfoSource             *string  `json:"info_source"`
	Coordinates            *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
}

func (h *Handler) CreateWaterRefillStation(c *gin.Context) {
	var in waterRefillStationCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isFree := false
	if in.IsFree != nil {
		isFree = *in.IsFree
	}
	accessible := false
	if in.Accessibility != nil {
		accessible = *in.Accessibility
	}
	var coordsJSON *string
	if in.Coordinates != nil {
		if b, err := json.Marshal(in.Coordinates); err == nil {
			s := string(b)
			coordsJSON = &s
		}
	}
	ctx := context.Background()
	var id string
	var created, updated int64
	err := h.pool.QueryRow(ctx, `insert into water_refill_stations(name,address,phone,water_type,opening_hours,is_free,container_required,daily_capacity,status,water_quality,facilities,accessibility,distance_to_disaster_area,notes,info_source,coordinates) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::text[],$12,$13,$14,$15,$16::jsonb) returning id,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
		in.Name, in.Address, in.Phone, in.WaterType, in.OpeningHours, isFree, in.ContainerRequired, in.DailyCapacity, in.Status, in.WaterQuality, in.Facilities, accessible, in.DistanceToDisasterArea, in.Notes, in.InfoSource, coordsJSON).Scan(&id, &created, &updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.WaterRefillStation{ID: id, Name: in.Name, Address: in.Address, Phone: in.Phone, WaterType: in.WaterType, OpeningHours: in.OpeningHours, IsFree: isFree, ContainerRequired: in.ContainerRequired, DailyCapacity: in.DailyCapacity, Status: in.Status, WaterQuality: in.WaterQuality, Facilities: in.Facilities, Accessibility: accessible, DistanceToDisasterArea: in.DistanceToDisasterArea, Notes: in.Notes, InfoSource: in.InfoSource, CreatedAt: created, UpdatedAt: updated}
	out.Coordinates = in.Coordinates
	c.JSON(http.StatusCreated, out)
}

type waterRefillStationPatchInput struct {
	Name                   *string   `json:"name"`
	Address                *string   `json:"address"`
	Phone                  *string   `json:"phone"`
	WaterType              *string   `json:"water_type"`
	OpeningHours           *string   `json:"opening_hours"`
	IsFree                 *bool     `json:"is_free"`
	ContainerRequired      *string   `json:"container_required"`
	DailyCapacity          *int      `json:"daily_capacity"`
	Status                 *string   `json:"status"`
	WaterQuality           *string   `json:"water_quality"`
	Facilities             *[]string `json:"facilities"`
	Accessibility          *bool     `json:"accessibility"`
	DistanceToDisasterArea *string   `json:"distance_to_disaster_area"`
	Notes                  *string   `json:"notes"`
	InfoSource             *string   `json:"info_source"`
	Coordinates            *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
}

func (h *Handler) PatchWaterRefillStation(c *gin.Context) {
	id := c.Param("id")
	var in waterRefillStationPatchInput
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
	if in.Name != nil {
		add("name=", *in.Name)
	}
	if in.Address != nil {
		add("address=", *in.Address)
	}
	if in.Phone != nil {
		add("phone=", *in.Phone)
	}
	if in.WaterType != nil {
		add("water_type=", *in.WaterType)
	}
	if in.OpeningHours != nil {
		add("opening_hours=", *in.OpeningHours)
	}
	if in.IsFree != nil {
		add("is_free=", *in.IsFree)
	}
	if in.ContainerRequired != nil {
		add("container_required=", *in.ContainerRequired)
	}
	if in.DailyCapacity != nil {
		add("daily_capacity=", *in.DailyCapacity)
	}
	if in.Status != nil {
		add("status=", *in.Status)
	}
	if in.WaterQuality != nil {
		add("water_quality=", *in.WaterQuality)
	}
	if in.Facilities != nil {
		add("facilities=", *in.Facilities)
	}
	if in.Accessibility != nil {
		add("accessibility=", *in.Accessibility)
	}
	if in.DistanceToDisasterArea != nil {
		add("distance_to_disaster_area=", *in.DistanceToDisasterArea)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.InfoSource != nil {
		add("info_source=", *in.InfoSource)
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
	query := "update water_refill_stations set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,name,address,phone,water_type,opening_hours,is_free,container_required,daily_capacity,status,water_quality,facilities,accessibility,distance_to_disaster_area,notes,info_source,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	row := h.pool.QueryRow(ctx, query, args...)
	var w models.WaterRefillStation
	var phone, containerReq, waterQuality, distance, notes, infoSource *string
	var dailyCap *int
	var facilities []string
	var isFree, accessibility bool
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&w.ID, &w.Name, &w.Address, &phone, &w.WaterType, &w.OpeningHours, &isFree, &containerReq, &dailyCap, &w.Status, &waterQuality, &facilities, &accessibility, &distance, &notes, &infoSource, &lat, &lng, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	w.Phone = phone
	w.ContainerRequired = containerReq
	w.WaterQuality = waterQuality
	w.DistanceToDisasterArea = distance
	w.Notes = notes
	w.InfoSource = infoSource
	w.DailyCapacity = dailyCap
	w.Facilities = facilities
	w.IsFree = isFree
	w.Accessibility = accessibility
	w.CreatedAt = created
	w.UpdatedAt = updated
	if lat != nil || lng != nil {
		w.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, w)
}

func (h *Handler) GetWaterRefillStation(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,name,address,phone,water_type,opening_hours,is_free,container_required,daily_capacity,status,water_quality,facilities,accessibility,distance_to_disaster_area,notes,info_source,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from water_refill_stations where id=$1`, id)
	var w models.WaterRefillStation
	var phone, containerReq, waterQuality, distance, notes, infoSource *string
	var dailyCap *int
	var facilities []string
	var isFree, accessibility bool
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&w.ID, &w.Name, &w.Address, &phone, &w.WaterType, &w.OpeningHours, &isFree, &containerReq, &dailyCap, &w.Status, &waterQuality, &facilities, &accessibility, &distance, &notes, &infoSource, &lat, &lng, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	w.Phone = phone
	w.ContainerRequired = containerReq
	w.WaterQuality = waterQuality
	w.DistanceToDisasterArea = distance
	w.Notes = notes
	w.InfoSource = infoSource
	w.DailyCapacity = dailyCap
	w.Facilities = facilities
	w.IsFree = isFree
	w.Accessibility = accessibility
	w.CreatedAt = created
	w.UpdatedAt = updated
	if lat != nil || lng != nil {
		w.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, w)
}

func (h *Handler) ListWaterRefillStations(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	status := c.Query("status")
	waterType := c.Query("water_type")
	isFree := c.Query("is_free")
	accessibility := c.Query("accessibility")
	ctx := context.Background()
	filters := []string{}
	args := []interface{}{}
	if status != "" {
		filters = append(filters, "status=$"+strconv.Itoa(len(args)+1))
		args = append(args, status)
	}
	if waterType != "" {
		filters = append(filters, "water_type=$"+strconv.Itoa(len(args)+1))
		args = append(args, waterType)
	}
	if isFree != "" {
		filters = append(filters, "is_free=$"+strconv.Itoa(len(args)+1))
		val := (isFree == "true" || isFree == "1")
		args = append(args, val)
	}
	if accessibility != "" {
		filters = append(filters, "accessibility=$"+strconv.Itoa(len(args)+1))
		val := (accessibility == "true" || accessibility == "1")
		args = append(args, val)
	}
	countQ := "select count(*) from water_refill_stations"
	dataQ := "select id,name,address,phone,water_type,opening_hours,is_free,container_required,daily_capacity,status,water_quality,facilities,accessibility,distance_to_disaster_area,notes,info_source,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from water_refill_stations"
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
	list := []models.WaterRefillStation{}
	for rows.Next() {
		var w models.WaterRefillStation
		var phone, containerReq, waterQuality, distance, notes, infoSource *string
		var dailyCap *int
		var facilities []string
		var free, acc bool
		var lat, lng *float64
		var created, updated int64
		if err := rows.Scan(&w.ID, &w.Name, &w.Address, &phone, &w.WaterType, &w.OpeningHours, &free, &containerReq, &dailyCap, &w.Status, &waterQuality, &facilities, &acc, &distance, &notes, &infoSource, &lat, &lng, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		w.Phone = phone
		w.ContainerRequired = containerReq
		w.WaterQuality = waterQuality
		w.DistanceToDisasterArea = distance
		w.Notes = notes
		w.InfoSource = infoSource
		w.DailyCapacity = dailyCap
		w.Facilities = facilities
		w.IsFree = free
		w.Accessibility = acc
		w.CreatedAt = created
		w.UpdatedAt = updated
		if lat != nil || lng != nil {
			w.Coordinates = &struct {
				Lat *float64 `json:"lat"`
				Lng *float64 `json:"lng"`
			}{Lat: lat, Lng: lng}
		}
		list = append(list, w)
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
