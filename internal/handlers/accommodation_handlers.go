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

type accommodationCreateInput struct {
	Township           string   `json:"township" binding:"required"`
	Name               string   `json:"name" binding:"required"`
	HasVacancy         string   `json:"has_vacancy" binding:"required"`
	AvailablePeriod    string   `json:"available_period" binding:"required"`
	Restrictions       *string  `json:"restrictions"`
	ContactInfo        string   `json:"contact_info" binding:"required"`
	RoomInfo           *string  `json:"room_info"`
	Address            string   `json:"address" binding:"required"`
	Pricing            string   `json:"pricing" binding:"required"`
	InfoSource         *string  `json:"info_source"`
	Notes              *string  `json:"notes"`
	Capacity           *int     `json:"capacity"`
	Status             string   `json:"status" binding:"required"`
	RegistrationMethod *string  `json:"registration_method"`
	Facilities         []string `json:"facilities"`
	DistanceToDisaster *string  `json:"distance_to_disaster_area"`
	Coordinates        *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
}

func (h *Handler) CreateAccommodation(c *gin.Context) {
	var in accommodationCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	var coordsJSON *string
	if in.Coordinates != nil {
		if b, err := json.Marshal(in.Coordinates); err == nil {
			s := string(b)
			coordsJSON = &s
		}
	}
	var id string
	var created, updated int64
	err := h.pool.QueryRow(ctx, `insert into accommodations(township,name,has_vacancy,available_period,restrictions,contact_info,room_info,address,pricing,info_source,notes,capacity,status,registration_method,facilities,distance_to_disaster_area,coordinates) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::text[],$16,$17::jsonb) returning id,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
		in.Township, in.Name, in.HasVacancy, in.AvailablePeriod, in.Restrictions, in.ContactInfo, in.RoomInfo, in.Address, in.Pricing, in.InfoSource, in.Notes, in.Capacity, in.Status, in.RegistrationMethod, in.Facilities, in.DistanceToDisaster, coordsJSON).Scan(&id, &created, &updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.Accommodation{ID: id, Township: in.Township, Name: in.Name, HasVacancy: in.HasVacancy, AvailablePeriod: in.AvailablePeriod, Restrictions: in.Restrictions, ContactInfo: in.ContactInfo, RoomInfo: in.RoomInfo, Address: in.Address, Pricing: in.Pricing, InfoSource: in.InfoSource, Notes: in.Notes, Capacity: in.Capacity, Status: in.Status, RegistrationMethod: in.RegistrationMethod, Facilities: in.Facilities, DistanceToDisasterArea: in.DistanceToDisaster, CreatedAt: created, UpdatedAt: updated}
	out.Coordinates = in.Coordinates
	c.JSON(http.StatusCreated, out)
}

type accommodationPatchInput struct {
	Township           *string   `json:"township"`
	Name               *string   `json:"name"`
	HasVacancy         *string   `json:"has_vacancy"`
	AvailablePeriod    *string   `json:"available_period"`
	Restrictions       *string   `json:"restrictions"`
	ContactInfo        *string   `json:"contact_info"`
	RoomInfo           *string   `json:"room_info"`
	Address            *string   `json:"address"`
	Pricing            *string   `json:"pricing"`
	InfoSource         *string   `json:"info_source"`
	Notes              *string   `json:"notes"`
	Capacity           *int      `json:"capacity"`
	Status             *string   `json:"status"`
	RegistrationMethod *string   `json:"registration_method"`
	Facilities         *[]string `json:"facilities"`
	DistanceToDisaster *string   `json:"distance_to_disaster_area"`
	Coordinates        *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
}

func (h *Handler) PatchAccommodation(c *gin.Context) {
	id := c.Param("id")
	var in accommodationPatchInput
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
	if in.Township != nil {
		add("township=", *in.Township)
	}
	if in.Name != nil {
		add("name=", *in.Name)
	}
	if in.HasVacancy != nil {
		add("has_vacancy=", *in.HasVacancy)
	}
	if in.AvailablePeriod != nil {
		add("available_period=", *in.AvailablePeriod)
	}
	if in.Restrictions != nil {
		add("restrictions=", *in.Restrictions)
	}
	if in.ContactInfo != nil {
		add("contact_info=", *in.ContactInfo)
	}
	if in.RoomInfo != nil {
		add("room_info=", *in.RoomInfo)
	}
	if in.Address != nil {
		add("address=", *in.Address)
	}
	if in.Pricing != nil {
		add("pricing=", *in.Pricing)
	}
	if in.InfoSource != nil {
		add("info_source=", *in.InfoSource)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.Capacity != nil {
		add("capacity=", *in.Capacity)
	}
	if in.Status != nil {
		add("status=", *in.Status)
	}
	if in.RegistrationMethod != nil {
		add("registration_method=", *in.RegistrationMethod)
	}
	if in.Facilities != nil {
		add("facilities=", *in.Facilities)
	}
	if in.DistanceToDisaster != nil {
		add("distance_to_disaster_area=", *in.DistanceToDisaster)
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
	query := "update accommodations set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,township,name,has_vacancy,available_period,restrictions,contact_info,room_info,address,pricing,info_source,notes,capacity,status,registration_method,facilities,distance_to_disaster_area,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	row := h.pool.QueryRow(ctx, query, args...)
	var a models.Accommodation
	var restrictions, roomInfo, infoSource, notes, regMethod, distance *string
	var facilities []string
	var capacity *int
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&a.ID, &a.Township, &a.Name, &a.HasVacancy, &a.AvailablePeriod, &restrictions, &a.ContactInfo, &roomInfo, &a.Address, &a.Pricing, &infoSource, &notes, &capacity, &a.Status, &regMethod, &facilities, &distance, &lat, &lng, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	a.Restrictions = restrictions
	a.RoomInfo = roomInfo
	a.InfoSource = infoSource
	a.Notes = notes
	a.RegistrationMethod = regMethod
	a.DistanceToDisasterArea = distance
	a.Capacity = capacity
	a.Facilities = facilities
	a.CreatedAt = created
	a.UpdatedAt = updated
	if lat != nil || lng != nil {
		a.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) GetAccommodation(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,township,name,has_vacancy,available_period,restrictions,contact_info,room_info,address,pricing,info_source,notes,capacity,status,registration_method,facilities,distance_to_disaster_area,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from accommodations where id=$1`, id)
	var a models.Accommodation
	var restrictions, roomInfo, infoSource, notes, regMethod, distance *string
	var facilities []string
	var capacity *int
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&a.ID, &a.Township, &a.Name, &a.HasVacancy, &a.AvailablePeriod, &restrictions, &a.ContactInfo, &roomInfo, &a.Address, &a.Pricing, &infoSource, &notes, &capacity, &a.Status, &regMethod, &facilities, &distance, &lat, &lng, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	a.Restrictions = restrictions
	a.RoomInfo = roomInfo
	a.InfoSource = infoSource
	a.Notes = notes
	a.RegistrationMethod = regMethod
	a.DistanceToDisasterArea = distance
	a.Capacity = capacity
	a.Facilities = facilities
	a.CreatedAt = created
	a.UpdatedAt = updated
	if lat != nil || lng != nil {
		a.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) ListAccommodations(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	status := c.Query("status")
	township := c.Query("township")
	hasVacancy := c.Query("has_vacancy")
	ctx := context.Background()
	filters := []string{}
	args := []interface{}{}
	if status != "" {
		filters = append(filters, "status=$"+strconv.Itoa(len(args)+1))
		args = append(args, status)
	}
	if township != "" {
		filters = append(filters, "township=$"+strconv.Itoa(len(args)+1))
		args = append(args, township)
	}
	if hasVacancy != "" {
		filters = append(filters, "has_vacancy=$"+strconv.Itoa(len(args)+1))
		args = append(args, hasVacancy)
	}
	countQ := "select count(*) from accommodations"
	dataQ := "select id,township,name,has_vacancy,available_period,restrictions,contact_info,room_info,address,pricing,info_source,notes,capacity,status,registration_method,facilities,distance_to_disaster_area,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from accommodations"
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
	list := []models.Accommodation{}
	for rows.Next() {
		var a models.Accommodation
		var restrictions, roomInfo, infoSource, notes, regMethod, distance *string
		var facilities []string
		var capacity *int
		var lat, lng *float64
		var created, updated int64
		if err := rows.Scan(&a.ID, &a.Township, &a.Name, &a.HasVacancy, &a.AvailablePeriod, &restrictions, &a.ContactInfo, &roomInfo, &a.Address, &a.Pricing, &infoSource, &notes, &capacity, &a.Status, &regMethod, &facilities, &distance, &lat, &lng, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		a.Restrictions = restrictions
		a.RoomInfo = roomInfo
		a.InfoSource = infoSource
		a.Notes = notes
		a.RegistrationMethod = regMethod
		a.DistanceToDisasterArea = distance
		a.Capacity = capacity
		a.Facilities = facilities
		a.CreatedAt = created
		a.UpdatedAt = updated
		if lat != nil || lng != nil {
			a.Coordinates = &struct {
				Lat *float64 `json:"lat"`
				Lng *float64 `json:"lng"`
			}{Lat: lat, Lng: lng}
		}
		list = append(list, a)
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
