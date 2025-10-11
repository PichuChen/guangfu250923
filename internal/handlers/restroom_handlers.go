package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"guangfu250923/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type restroomCreateInput struct {
	Name                   string   `json:"name" binding:"required"`
	Address                string   `json:"address" binding:"required"`
	Phone                  *string  `json:"phone"`
	FacilityType           string   `json:"facility_type" binding:"required"`
	OpeningHours           string   `json:"opening_hours" binding:"required"`
	IsFree                 *bool    `json:"is_free" binding:"required"`
	MaleUnits              *int     `json:"male_units"`
	FemaleUnits            *int     `json:"female_units"`
	UnisexUnits            *int     `json:"unisex_units"`
	AccessibleUnits        *int     `json:"accessible_units"`
	HasWater               *bool    `json:"has_water" binding:"required"`
	HasLighting            *bool    `json:"has_lighting" binding:"required"`
	Status                 string   `json:"status" binding:"required"`
	Cleanliness            *string  `json:"cleanliness"`
	LastCleaned            *int64   `json:"last_cleaned"`
	Facilities             []string `json:"facilities"`
	DistanceToDisasterArea *string  `json:"distance_to_disaster_area"`
	Notes                  *string  `json:"notes"`
	InfoSource             *string  `json:"info_source"`
	Coordinates            *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
}

func (h *Handler) CreateRestroom(c *gin.Context) {
	var in restroomCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isFree := false
	if in.IsFree != nil {
		isFree = *in.IsFree
	}
	hasWater := false
	if in.HasWater != nil {
		hasWater = *in.HasWater
	}
	hasLighting := false
	if in.HasLighting != nil {
		hasLighting = *in.HasLighting
	}
	var coordsJSON *string
	if in.Coordinates != nil {
		if b, err := json.Marshal(in.Coordinates); err == nil {
			s := string(b)
			coordsJSON = &s
		}
	}
	var lastCleaned *time.Time
	if in.LastCleaned != nil {
		t := time.Unix(*in.LastCleaned, 0)
		lastCleaned = &t
	}
	ctx := context.Background()
	var id string
	var created, updated int64
	err := h.pool.QueryRow(ctx, `insert into restrooms(name,address,phone,facility_type,opening_hours,is_free,male_units,female_units,unisex_units,accessible_units,has_water,has_lighting,status,cleanliness,last_cleaned,facilities,distance_to_disaster_area,notes,info_source,coordinates) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::text[],$17,$18,$19,$20::jsonb) returning id,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
		in.Name, in.Address, in.Phone, in.FacilityType, in.OpeningHours, isFree, in.MaleUnits, in.FemaleUnits, in.UnisexUnits, in.AccessibleUnits, hasWater, hasLighting, in.Status, in.Cleanliness, lastCleaned, in.Facilities, in.DistanceToDisasterArea, in.Notes, in.InfoSource, coordsJSON).Scan(&id, &created, &updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.Restroom{ID: id, Name: in.Name, Address: in.Address, Phone: in.Phone, FacilityType: in.FacilityType, OpeningHours: in.OpeningHours, IsFree: isFree, MaleUnits: in.MaleUnits, FemaleUnits: in.FemaleUnits, UnisexUnits: in.UnisexUnits, AccessibleUnits: in.AccessibleUnits, HasWater: hasWater, HasLighting: hasLighting, Status: in.Status, Cleanliness: in.Cleanliness, Facilities: in.Facilities, DistanceToDisasterArea: in.DistanceToDisasterArea, Notes: in.Notes, InfoSource: in.InfoSource, CreatedAt: created, UpdatedAt: updated}
	if lastCleaned != nil {
		ts := lastCleaned.Unix()
		out.LastCleaned = &ts
	}
	out.Coordinates = in.Coordinates
	c.JSON(http.StatusCreated, out)
}

type restroomPatchInput struct {
	Name                   *string   `json:"name"`
	Address                *string   `json:"address"`
	Phone                  *string   `json:"phone"`
	FacilityType           *string   `json:"facility_type"`
	OpeningHours           *string   `json:"opening_hours"`
	IsFree                 *bool     `json:"is_free"`
	MaleUnits              *int      `json:"male_units"`
	FemaleUnits            *int      `json:"female_units"`
	UnisexUnits            *int      `json:"unisex_units"`
	AccessibleUnits        *int      `json:"accessible_units"`
	HasWater               *bool     `json:"has_water"`
	HasLighting            *bool     `json:"has_lighting"`
	Status                 *string   `json:"status"`
	Cleanliness            *string   `json:"cleanliness"`
	LastCleaned            *int64    `json:"last_cleaned"`
	Facilities             *[]string `json:"facilities"`
	DistanceToDisasterArea *string   `json:"distance_to_disaster_area"`
	Notes                  *string   `json:"notes"`
	InfoSource             *string   `json:"info_source"`
	Coordinates            *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
}

func (h *Handler) PatchRestroom(c *gin.Context) {
	id := c.Param("id")
	var in restroomPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	setParts := []string{}
	args := []interface{}{}
	idx := 1
	add := func(col string, val interface{}) {
		setParts = append(setParts, col+"$"+strconv.Itoa(idx))
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
	if in.FacilityType != nil {
		add("facility_type=", *in.FacilityType)
	}
	if in.OpeningHours != nil {
		add("opening_hours=", *in.OpeningHours)
	}
	if in.IsFree != nil {
		add("is_free=", *in.IsFree)
	}
	if in.MaleUnits != nil {
		add("male_units=", *in.MaleUnits)
	}
	if in.FemaleUnits != nil {
		add("female_units=", *in.FemaleUnits)
	}
	if in.UnisexUnits != nil {
		add("unisex_units=", *in.UnisexUnits)
	}
	if in.AccessibleUnits != nil {
		add("accessible_units=", *in.AccessibleUnits)
	}
	if in.HasWater != nil {
		add("has_water=", *in.HasWater)
	}
	if in.HasLighting != nil {
		add("has_lighting=", *in.HasLighting)
	}
	if in.Status != nil {
		add("status=", *in.Status)
	}
	if in.Cleanliness != nil {
		add("cleanliness=", *in.Cleanliness)
	}
	if in.LastCleaned != nil {
		t := time.Unix(*in.LastCleaned, 0)
		add("last_cleaned=", t)
	}
	if in.Facilities != nil {
		add("facilities=", *in.Facilities)
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
	query := "update restrooms set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,name,address,phone,facility_type,opening_hours,is_free,male_units,female_units,unisex_units,accessible_units,has_water,has_lighting,status,cleanliness,extract(epoch from last_cleaned)::bigint,facilities,distance_to_disaster_area,notes,info_source,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	row := h.pool.QueryRow(ctx, query, args...)
	var r models.Restroom
	var phone, cleanliness, distance, notes, infoSource *string
	var male, female, unisex, accessible *int
	var lastCleaned *int64
	var facilities []string
	var isFree, hasWater, hasLighting bool
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&r.ID, &r.Name, &r.Address, &phone, &r.FacilityType, &r.OpeningHours, &isFree, &male, &female, &unisex, &accessible, &hasWater, &hasLighting, &r.Status, &cleanliness, &lastCleaned, &facilities, &distance, &notes, &infoSource, &lat, &lng, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.Phone = phone
	r.Cleanliness = cleanliness
	r.DistanceToDisasterArea = distance
	r.Notes = notes
	r.InfoSource = infoSource
	r.MaleUnits = male
	r.FemaleUnits = female
	r.UnisexUnits = unisex
	r.AccessibleUnits = accessible
	r.Facilities = facilities
	r.IsFree = isFree
	r.HasWater = hasWater
	r.HasLighting = hasLighting
	r.CreatedAt = created
	r.UpdatedAt = updated
	r.LastCleaned = lastCleaned
	if lat != nil || lng != nil {
		r.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handler) GetRestroom(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,name,address,phone,facility_type,opening_hours,is_free,male_units,female_units,unisex_units,accessible_units,has_water,has_lighting,status,cleanliness,extract(epoch from last_cleaned)::bigint,facilities,distance_to_disaster_area,notes,info_source,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from restrooms where id=$1`, id)
	var r models.Restroom
	var phone, cleanliness, distance, notes, infoSource *string
	var male, female, unisex, accessible *int
	var lastCleaned *int64
	var facilities []string
	var isFree, hasWater, hasLighting bool
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&r.ID, &r.Name, &r.Address, &phone, &r.FacilityType, &r.OpeningHours, &isFree, &male, &female, &unisex, &accessible, &hasWater, &hasLighting, &r.Status, &cleanliness, &lastCleaned, &facilities, &distance, &notes, &infoSource, &lat, &lng, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.Phone = phone
	r.Cleanliness = cleanliness
	r.DistanceToDisasterArea = distance
	r.Notes = notes
	r.InfoSource = infoSource
	r.MaleUnits = male
	r.FemaleUnits = female
	r.UnisexUnits = unisex
	r.AccessibleUnits = accessible
	r.Facilities = facilities
	r.IsFree = isFree
	r.HasWater = hasWater
	r.HasLighting = hasLighting
	r.CreatedAt = created
	r.UpdatedAt = updated
	r.LastCleaned = lastCleaned
	if lat != nil || lng != nil {
		r.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handler) ListRestrooms(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	status := c.Query("status")
	facilityType := c.Query("facility_type")
	isFree := c.Query("is_free")
	hasWater := c.Query("has_water")
	hasLighting := c.Query("has_lighting")
	ctx := context.Background()
	filters := []string{}
	args := []interface{}{}
	if status != "" {
		filters = append(filters, "status=$"+strconv.Itoa(len(args)+1))
		args = append(args, status)
	}
	if facilityType != "" {
		filters = append(filters, "facility_type=$"+strconv.Itoa(len(args)+1))
		args = append(args, facilityType)
	}
	if isFree != "" {
		filters = append(filters, "is_free=$"+strconv.Itoa(len(args)+1))
		args = append(args, isFree == "true" || isFree == "1")
	}
	if hasWater != "" {
		filters = append(filters, "has_water=$"+strconv.Itoa(len(args)+1))
		args = append(args, hasWater == "true" || hasWater == "1")
	}
	if hasLighting != "" {
		filters = append(filters, "has_lighting=$"+strconv.Itoa(len(args)+1))
		args = append(args, hasLighting == "true" || hasLighting == "1")
	}
	countQ := "select count(*) from restrooms"
	dataQ := "select id,name,address,phone,facility_type,opening_hours,is_free,male_units,female_units,unisex_units,accessible_units,has_water,has_lighting,status,cleanliness,extract(epoch from last_cleaned)::bigint,facilities,distance_to_disaster_area,notes,info_source,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from restrooms"
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
	list := []models.Restroom{}
	for rows.Next() {
		var r models.Restroom
		var phone, cleanliness, distance, notes, infoSource *string
		var male, female, unisex, accessible *int
		var lastCleaned *int64
		var facilities []string
		var free, water, lighting bool
		var lat, lng *float64
		var created, updated int64
		if err := rows.Scan(&r.ID, &r.Name, &r.Address, &phone, &r.FacilityType, &r.OpeningHours, &free, &male, &female, &unisex, &accessible, &water, &lighting, &r.Status, &cleanliness, &lastCleaned, &facilities, &distance, &notes, &infoSource, &lat, &lng, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		r.Phone = phone
		r.Cleanliness = cleanliness
		r.DistanceToDisasterArea = distance
		r.Notes = notes
		r.InfoSource = infoSource
		r.MaleUnits = male
		r.FemaleUnits = female
		r.UnisexUnits = unisex
		r.AccessibleUnits = accessible
		r.Facilities = facilities
		r.IsFree = free
		r.HasWater = water
		r.HasLighting = lighting
		r.CreatedAt = created
		r.UpdatedAt = updated
		r.LastCleaned = lastCleaned
		if lat != nil || lng != nil {
			r.Coordinates = &struct {
				Lat *float64 `json:"lat"`
				Lng *float64 `json:"lng"`
			}{Lat: lat, Lng: lng}
		}
		list = append(list, r)
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
