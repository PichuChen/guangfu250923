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

type showerStationCreateInput struct {
	Name           string  `json:"name" binding:"required"`
	Address        string  `json:"address" binding:"required"`
	Phone          *string `json:"phone"`
	FacilityType   string  `json:"facility_type" binding:"required"`
	TimeSlots      string  `json:"time_slots" binding:"required"`
	GenderSchedule *struct {
		Male   []string `json:"male"`
		Female []string `json:"female"`
	} `json:"gender_schedule"`
	AvailablePeriod     string   `json:"available_period" binding:"required"`
	Capacity            *int     `json:"capacity"`
	IsFree              *bool    `json:"is_free" binding:"required"`
	Pricing             *string  `json:"pricing"`
	Notes               *string  `json:"notes"`
	InfoSource          *string  `json:"info_source"`
	Status              string   `json:"status" binding:"required"`
	Facilities          []string `json:"facilities"`
	DistanceToGuangfu   *string  `json:"distance_to_guangfu"`
	RequiresAppointment *bool    `json:"requires_appointment" binding:"required"`
	ContactMethod       *string  `json:"contact_method"`
	Coordinates         *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
}

func (h *Handler) CreateShowerStation(c *gin.Context) {
	var in showerStationCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	isFree := false
	if in.IsFree != nil {
		isFree = *in.IsFree
	}
	reqApp := false
	if in.RequiresAppointment != nil {
		reqApp = *in.RequiresAppointment
	}
	var coordsJSON *string
	if in.Coordinates != nil {
		if b, err := json.Marshal(in.Coordinates); err == nil {
			s := string(b)
			coordsJSON = &s
		}
	}
	var genderJSON []byte
	if in.GenderSchedule != nil {
		genderJSON, _ = json.Marshal(in.GenderSchedule)
	}
	var id string
	var created, updated int64
	err := h.pool.QueryRow(ctx, `insert into shower_stations(name,address,phone,facility_type,time_slots,gender_schedule,available_period,capacity,is_free,pricing,notes,info_source,status,facilities,distance_to_guangfu,requires_appointment,contact_method,coordinates) values($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11,$12,$13,$14::text[],$15,$16,$17,$18::jsonb) returning id,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
		in.Name, in.Address, in.Phone, in.FacilityType, in.TimeSlots, genderJSON, in.AvailablePeriod, in.Capacity, isFree, in.Pricing, in.Notes, in.InfoSource, in.Status, in.Facilities, in.DistanceToGuangfu, reqApp, in.ContactMethod, coordsJSON).Scan(&id, &created, &updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.ShowerStation{ID: id, Name: in.Name, Address: in.Address, Phone: in.Phone, FacilityType: in.FacilityType, TimeSlots: in.TimeSlots, AvailablePeriod: in.AvailablePeriod, Capacity: in.Capacity, IsFree: isFree, Pricing: in.Pricing, Notes: in.Notes, InfoSource: in.InfoSource, Status: in.Status, Facilities: in.Facilities, DistanceToGuangfu: in.DistanceToGuangfu, RequiresAppointment: reqApp, ContactMethod: in.ContactMethod, CreatedAt: created, UpdatedAt: updated}
	if in.GenderSchedule != nil {
		out.GenderSchedule = &struct {
			Male   []string `json:"male"`
			Female []string `json:"female"`
		}{Male: in.GenderSchedule.Male, Female: in.GenderSchedule.Female}
	}
	out.Coordinates = in.Coordinates
	c.JSON(http.StatusCreated, out)
}

type showerStationPatchInput struct {
	Name           *string `json:"name"`
	Address        *string `json:"address"`
	Phone          *string `json:"phone"`
	FacilityType   *string `json:"facility_type"`
	TimeSlots      *string `json:"time_slots"`
	GenderSchedule *struct {
		Male   []string `json:"male"`
		Female []string `json:"female"`
	} `json:"gender_schedule"`
	AvailablePeriod     *string   `json:"available_period"`
	Capacity            *int      `json:"capacity"`
	IsFree              *bool     `json:"is_free"`
	Pricing             *string   `json:"pricing"`
	Notes               *string   `json:"notes"`
	InfoSource          *string   `json:"info_source"`
	Status              *string   `json:"status"`
	Facilities          *[]string `json:"facilities"`
	DistanceToGuangfu   *string   `json:"distance_to_guangfu"`
	RequiresAppointment *bool     `json:"requires_appointment"`
	ContactMethod       *string   `json:"contact_method"`
	Coordinates         *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
}

func (h *Handler) PatchShowerStation(c *gin.Context) {
	id := c.Param("id")
	var in showerStationPatchInput
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
	if in.FacilityType != nil {
		add("facility_type=", *in.FacilityType)
	}
	if in.TimeSlots != nil {
		add("time_slots=", *in.TimeSlots)
	}
	if in.GenderSchedule != nil {
		js, _ := json.Marshal(in.GenderSchedule)
		add("gender_schedule=", js)
	}
	if in.AvailablePeriod != nil {
		add("available_period=", *in.AvailablePeriod)
	}
	if in.Capacity != nil {
		add("capacity=", *in.Capacity)
	}
	if in.IsFree != nil {
		add("is_free=", *in.IsFree)
	}
	if in.Pricing != nil {
		add("pricing=", *in.Pricing)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.InfoSource != nil {
		add("info_source=", *in.InfoSource)
	}
	if in.Status != nil {
		add("status=", *in.Status)
	}
	if in.Facilities != nil {
		add("facilities=", *in.Facilities)
	}
	if in.DistanceToGuangfu != nil {
		add("distance_to_guangfu=", *in.DistanceToGuangfu)
	}
	if in.RequiresAppointment != nil {
		add("requires_appointment=", *in.RequiresAppointment)
	}
	if in.ContactMethod != nil {
		add("contact_method=", *in.ContactMethod)
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
	query := "update shower_stations set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,name,address,phone,facility_type,time_slots,gender_schedule,available_period,capacity,is_free,pricing,notes,info_source,status,facilities,distance_to_guangfu,requires_appointment,contact_method,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	row := h.pool.QueryRow(ctx, query, args...)
	var s models.ShowerStation
	var phone, pricing, notes, infoSource, distance, contactMethod *string
	var genderJSON []byte
	var facilities []string
	var capacity *int
	var isFree bool
	var reqApp bool
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&s.ID, &s.Name, &s.Address, &phone, &s.FacilityType, &s.TimeSlots, &genderJSON, &s.AvailablePeriod, &capacity, &isFree, &pricing, &notes, &infoSource, &s.Status, &facilities, &distance, &reqApp, &contactMethod, &lat, &lng, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.Phone = phone
	s.Pricing = pricing
	s.Notes = notes
	s.InfoSource = infoSource
	s.DistanceToGuangfu = distance
	s.ContactMethod = contactMethod
	s.Capacity = capacity
	s.IsFree = isFree
	s.RequiresAppointment = reqApp
	s.Facilities = facilities
	s.CreatedAt = created
	s.UpdatedAt = updated
	if len(genderJSON) > 0 {
		var gs struct {
			Male   []string `json:"male"`
			Female []string `json:"female"`
		}
		_ = json.Unmarshal(genderJSON, &gs)
		s.GenderSchedule = &gs
	}
	if lat != nil || lng != nil {
		s.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) GetShowerStation(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,name,address,phone,facility_type,time_slots,gender_schedule,available_period,capacity,is_free,pricing,notes,info_source,status,facilities,distance_to_guangfu,requires_appointment,contact_method,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from shower_stations where id=$1`, id)
	var s models.ShowerStation
	var phone, pricing, notes, infoSource, distance, contactMethod *string
	var genderJSON []byte
	var facilities []string
	var capacity *int
	var isFree bool
	var reqApp bool
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&s.ID, &s.Name, &s.Address, &phone, &s.FacilityType, &s.TimeSlots, &genderJSON, &s.AvailablePeriod, &capacity, &isFree, &pricing, &notes, &infoSource, &s.Status, &facilities, &distance, &reqApp, &contactMethod, &lat, &lng, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.Phone = phone
	s.Pricing = pricing
	s.Notes = notes
	s.InfoSource = infoSource
	s.DistanceToGuangfu = distance
	s.ContactMethod = contactMethod
	s.Capacity = capacity
	s.IsFree = isFree
	s.RequiresAppointment = reqApp
	s.Facilities = facilities
	s.CreatedAt = created
	s.UpdatedAt = updated
	if len(genderJSON) > 0 {
		var gs struct {
			Male   []string `json:"male"`
			Female []string `json:"female"`
		}
		_ = json.Unmarshal(genderJSON, &gs)
		s.GenderSchedule = &gs
	}
	if lat != nil || lng != nil {
		s.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) ListShowerStations(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	status := c.Query("status")
	facilityType := c.Query("facility_type")
	isFree := c.Query("is_free")
	requiresApp := c.Query("requires_appointment")
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
		val := (isFree == "true" || isFree == "1")
		args = append(args, val)
	}
	if requiresApp != "" {
		filters = append(filters, "requires_appointment=$"+strconv.Itoa(len(args)+1))
		val := (requiresApp == "true" || requiresApp == "1")
		args = append(args, val)
	}
	countQ := "select count(*) from shower_stations"
	dataQ := "select id,name,address,phone,facility_type,time_slots,gender_schedule,available_period,capacity,is_free,pricing,notes,info_source,status,facilities,distance_to_guangfu,requires_appointment,contact_method,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from shower_stations"
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
	list := []models.ShowerStation{}
	for rows.Next() {
		var s models.ShowerStation
		var phone, pricing, notes, infoSource, distance, contactMethod *string
		var genderJSON []byte
		var facilities []string
		var capacity *int
		var free bool
		var reqApp bool
		var lat, lng *float64
		var created, updated int64
		if err := rows.Scan(&s.ID, &s.Name, &s.Address, &phone, &s.FacilityType, &s.TimeSlots, &genderJSON, &s.AvailablePeriod, &capacity, &free, &pricing, &notes, &infoSource, &s.Status, &facilities, &distance, &reqApp, &contactMethod, &lat, &lng, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.Phone = phone
		s.Pricing = pricing
		s.Notes = notes
		s.InfoSource = infoSource
		s.DistanceToGuangfu = distance
		s.ContactMethod = contactMethod
		s.Capacity = capacity
		s.IsFree = free
		s.RequiresAppointment = reqApp
		s.Facilities = facilities
		s.CreatedAt = created
		s.UpdatedAt = updated
		if len(genderJSON) > 0 {
			var gs struct {
				Male   []string `json:"male"`
				Female []string `json:"female"`
			}
			_ = json.Unmarshal(genderJSON, &gs)
			s.GenderSchedule = &gs
		}
		if lat != nil || lng != nil {
			s.Coordinates = &struct {
				Lat *float64 `json:"lat"`
				Lng *float64 `json:"lng"`
			}{Lat: lat, Lng: lng}
		}
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
	c.JSON(http.StatusOK, gin.H{"@context": "https://www.w3.org/ns/hydra/context.jsonld", "@type": "Collection", "totalItems": total, "member": list, "limit": limit, "offset": offset, "next": next, "previous": prev})
}
