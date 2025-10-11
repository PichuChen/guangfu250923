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

type medicalStationCreateInput struct {
	StationType     string   `json:"station_type" binding:"required"`
	Name            string   `json:"name" binding:"required"`
	Location        string   `json:"location"`
	DetailedAddress *string  `json:"detailed_address"`
	Phone           *string  `json:"phone"`
	ContactPerson   *string  `json:"contact_person"`
	Status          string   `json:"status" binding:"required"`
	Services        []string `json:"services"`
	OperatingHours  *string  `json:"operating_hours"`
	Equipment       []string `json:"equipment"`
	MedicalStaff    *int     `json:"medical_staff"`
	DailyCapacity   *int     `json:"daily_capacity"`
	Coordinates     *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	AffiliatedOrganization *string `json:"affiliated_organization"`
	Notes                  *string `json:"notes"`
	Link                   *string `json:"link"`
}

func (h *Handler) CreateMedicalStation(c *gin.Context) {
	var in medicalStationCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.Status == "" {
		in.Status = "active"
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
	err := h.pool.QueryRow(ctx, `insert into medical_stations(station_type,name,location,detailed_address,phone,contact_person,status,services,equipment,operating_hours,medical_staff,daily_capacity,affiliated_organization,notes,link,coordinates) values($1,$2,$3,$4,$5,$6,$7,$8::text[],$9::text[],$10,$11,$12,$13,$14,$15,$16::jsonb) returning id,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
		in.StationType, in.Name, in.Location, in.DetailedAddress, in.Phone, in.ContactPerson, in.Status, in.Services, in.Equipment, in.OperatingHours, in.MedicalStaff, in.DailyCapacity, in.AffiliatedOrganization, in.Notes, in.Link, coordsJSON).Scan(&id, &created, &updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.MedicalStation{ID: id, StationType: in.StationType, Name: in.Name, Location: in.Location, DetailedAddress: in.DetailedAddress, Phone: in.Phone, ContactPerson: in.ContactPerson, Status: in.Status, Services: in.Services, Equipment: in.Equipment, OperatingHours: in.OperatingHours, MedicalStaff: in.MedicalStaff, DailyCapacity: in.DailyCapacity, AffiliatedOrganization: in.AffiliatedOrganization, Notes: in.Notes, Link: in.Link, CreatedAt: created, UpdatedAt: updated}
	out.Coordinates = in.Coordinates
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) ListMedicalStations(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	status := c.Query("status")
	stationType := c.Query("station_type")
	ctx := context.Background()

	// Build filters
	filters := []string{}
	args := []interface{}{}
	if status != "" {
		filters = append(filters, "status=$"+strconv.Itoa(len(args)+1))
		args = append(args, status)
	}
	if stationType != "" {
		filters = append(filters, "station_type=$"+strconv.Itoa(len(args)+1))
		args = append(args, stationType)
	}

	countQuery := "select count(*) from medical_stations"
	dataQuery := "select id,station_type,name,location,detailed_address,phone,contact_person,status,services,equipment,operating_hours,medical_staff,daily_capacity,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,affiliated_organization,notes,link,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from medical_stations"
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

	argsWithPage := append(args, limit, offset)
	dataQuery += " order by updated_at desc limit $" + strconv.Itoa(len(args)+1) + " offset $" + strconv.Itoa(len(args)+2)

	rows, err := h.pool.Query(ctx, dataQuery, argsWithPage...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []models.MedicalStation{}
	for rows.Next() {
		var m models.MedicalStation
		var detailedAddr, phone, contactPerson, operatingHours, affiliatedOrg, notes, link *string
		var medStaff, dailyCap *int
		var services, equipment []string
		var lat, lng *float64
		var created, updated int64
	if err := rows.Scan(&m.ID, &m.StationType, &m.Name, &m.Location, &detailedAddr, &phone, &contactPerson, &m.Status, &services, &equipment, &operatingHours, &medStaff, &dailyCap, &lat, &lng, &affiliatedOrg, &notes, &link, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		m.DetailedAddress = detailedAddr
		m.Phone = phone
		m.ContactPerson = contactPerson
		m.OperatingHours = operatingHours
		m.AffiliatedOrganization = affiliatedOrg
		m.Notes = notes
		m.Link = link
		m.MedicalStaff = medStaff
		m.DailyCapacity = dailyCap
		m.Services = services
		m.Equipment = equipment
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

type medicalStationPatchInput struct {
	StationType     *string   `json:"station_type"`
	Name            *string   `json:"name"`
	Location        *string   `json:"location"`
	DetailedAddress *string   `json:"detailed_address"`
	Phone           *string   `json:"phone"`
	ContactPerson   *string   `json:"contact_person"`
	Status          *string   `json:"status"`
	Services        *[]string `json:"services"`
	OperatingHours  *string   `json:"operating_hours"`
	Equipment       *[]string `json:"equipment"`
	MedicalStaff    *int      `json:"medical_staff"`
	DailyCapacity   *int      `json:"daily_capacity"`
	Coordinates     *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	AffiliatedOrganization *string `json:"affiliated_organization"`
	Notes                  *string `json:"notes"`
	Link                   *string `json:"link"`
}

func (h *Handler) PatchMedicalStation(c *gin.Context) {
	id := c.Param("id")
	var in medicalStationPatchInput
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
	if in.StationType != nil {
		add("station_type=", *in.StationType)
	}
	if in.Name != nil {
		add("name=", *in.Name)
	}
	if in.Location != nil {
		add("location=", *in.Location)
	}
	if in.DetailedAddress != nil {
		add("detailed_address=", *in.DetailedAddress)
	}
	if in.Phone != nil {
		add("phone=", *in.Phone)
	}
	if in.ContactPerson != nil {
		add("contact_person=", *in.ContactPerson)
	}
	if in.Status != nil {
		add("status=", *in.Status)
	}
	if in.Services != nil {
		add("services=", *in.Services)
	}
	if in.Equipment != nil {
		add("equipment=", *in.Equipment)
	}
	if in.OperatingHours != nil {
		add("operating_hours=", *in.OperatingHours)
	}
	if in.MedicalStaff != nil {
		add("medical_staff=", *in.MedicalStaff)
	}
	if in.DailyCapacity != nil {
		add("daily_capacity=", *in.DailyCapacity)
	}
	if in.AffiliatedOrganization != nil {
		add("affiliated_organization=", *in.AffiliatedOrganization)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.Link != nil {
		add("link=", *in.Link)
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
	query := "update medical_stations set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,station_type,name,location,detailed_address,phone,contact_person,status,services,equipment,operating_hours,medical_staff,daily_capacity,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,affiliated_organization,notes,link,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	row := h.pool.QueryRow(ctx, query, args...)
	var m models.MedicalStation
	var detailedAddr, phone, contactPerson, operatingHours, affiliatedOrg, notes, link *string
	var medStaff, dailyCap *int
	var services, equipment []string
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&m.ID, &m.StationType, &m.Name, &m.Location, &detailedAddr, &phone, &contactPerson, &m.Status, &services, &equipment, &operatingHours, &medStaff, &dailyCap, &lat, &lng, &affiliatedOrg, &notes, &link, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	m.DetailedAddress = detailedAddr
	m.Phone = phone
	m.ContactPerson = contactPerson
	m.OperatingHours = operatingHours
	m.AffiliatedOrganization = affiliatedOrg
	m.Notes = notes
	m.Link = link
	m.MedicalStaff = medStaff
	m.DailyCapacity = dailyCap
	m.Services = services
	m.Equipment = equipment
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

func (h *Handler) GetMedicalStation(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,station_type,name,location,detailed_address,phone,contact_person,status,services,equipment,operating_hours,medical_staff,daily_capacity,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,affiliated_organization,notes,link,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from medical_stations where id=$1`, id)
	var m models.MedicalStation
	var detailedAddr, phone, contactPerson, operatingHours, affiliatedOrg, notes, link *string
	var medStaff, dailyCap *int
	var services, equipment []string
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&m.ID, &m.StationType, &m.Name, &m.Location, &detailedAddr, &phone, &contactPerson, &m.Status, &services, &equipment, &operatingHours, &medStaff, &dailyCap, &lat, &lng, &affiliatedOrg, &notes, &link, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	m.DetailedAddress = detailedAddr
	m.Phone = phone
	m.ContactPerson = contactPerson
	m.OperatingHours = operatingHours
	m.AffiliatedOrganization = affiliatedOrg
	m.Notes = notes
	m.Link = link
	m.MedicalStaff = medStaff
	m.DailyCapacity = dailyCap
	m.Services = services
	m.Equipment = equipment
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
