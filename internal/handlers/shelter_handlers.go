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

type shelterCreateInput struct {
	Name             string   `json:"name" binding:"required"`
	Location         string   `json:"location" binding:"required"`
	Phone            string   `json:"phone" binding:"required"`
	Link             *string  `json:"link"`
	Status           string   `json:"status" binding:"required"`
	Capacity         *int     `json:"capacity"`
	CurrentOccupancy *int     `json:"current_occupancy"`
	AvailableSpaces  *int     `json:"available_spaces"`
	Facilities       []string `json:"facilities"`
	ContactPerson    *string  `json:"contact_person"`
	Notes            *string  `json:"notes"`
	Coordinates      *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	OpeningHours *string `json:"opening_hours"`
}

func (h *Handler) CreateShelter(c *gin.Context) {
	var in shelterCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.Status == "" {
		in.Status = "open"
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
	err := h.pool.QueryRow(ctx, `insert into shelters(name,location,phone,link,status,capacity,current_occupancy,available_spaces,facilities,contact_person,notes,opening_hours,coordinates) values($1,$2,$3,$4,$5,$6,$7,$8,$9::text[],$10,$11,$12,$13::jsonb) returning id,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
		in.Name, in.Location, in.Phone, in.Link, in.Status, in.Capacity, in.CurrentOccupancy, in.AvailableSpaces, in.Facilities, in.ContactPerson, in.Notes, in.OpeningHours, coordsJSON).Scan(&id, &created, &updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.Shelter{ID: id, Name: in.Name, Location: in.Location, Phone: in.Phone, Link: in.Link, Status: in.Status, Capacity: in.Capacity, CurrentOccupancy: in.CurrentOccupancy, AvailableSpaces: in.AvailableSpaces, Facilities: in.Facilities, ContactPerson: in.ContactPerson, Notes: in.Notes, OpeningHours: in.OpeningHours, CreatedAt: created, UpdatedAt: updated}
	out.Coordinates = in.Coordinates
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) ListShelters(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	status := c.Query("status")
	ctx := context.Background()
	var total int
	if status != "" {
		h.pool.QueryRow(ctx, `select count(*) from shelters where status=$1`, status).Scan(&total)
	} else {
		h.pool.QueryRow(ctx, `select count(*) from shelters`).Scan(&total)
	}
	base := `select id,name,location,phone,link,status,capacity,current_occupancy,available_spaces,facilities,contact_person,notes,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,opening_hours,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from shelters`
	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = h.pool.Query(ctx, base+` where status=$1 order by updated_at desc limit $2 offset $3`, status, limit, offset)
	} else {
		rows, err = h.pool.Query(ctx, base+` order by updated_at desc limit $1 offset $2`, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []models.Shelter{}
	for rows.Next() {
		var s models.Shelter
		var link, contactPerson, notes, opening *string
		var capacity, currentOcc, avail *int
		var facilities []string
		var lat, lng *float64
		var created, updated int64
		if err = rows.Scan(&s.ID, &s.Name, &s.Location, &s.Phone, &link, &s.Status, &capacity, &currentOcc, &avail, &facilities, &contactPerson, &notes, &lat, &lng, &opening, &created, &updated); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s.Link = link
		s.ContactPerson = contactPerson
		s.Notes = notes
		s.OpeningHours = opening
		s.Capacity = capacity
		s.CurrentOccupancy = currentOcc
		s.AvailableSpaces = avail
		s.Facilities = facilities
		s.CreatedAt = created
		s.UpdatedAt = updated
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

func (h *Handler) GetShelter(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,name,location,phone,link,status,capacity,current_occupancy,available_spaces,facilities,contact_person,notes,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,opening_hours,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from shelters where id=$1`, id)
	var s models.Shelter
	var link, contactPerson, notes, opening *string
	var capacity, currentOcc, avail *int
	var facilities []string
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&s.ID, &s.Name, &s.Location, &s.Phone, &link, &s.Status, &capacity, &currentOcc, &avail, &facilities, &contactPerson, &notes, &lat, &lng, &opening, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.Link = link
	s.ContactPerson = contactPerson
	s.Notes = notes
	s.OpeningHours = opening
	s.Capacity = capacity
	s.CurrentOccupancy = currentOcc
	s.AvailableSpaces = avail
	s.Facilities = facilities
	s.CreatedAt = created
	s.UpdatedAt = updated
	if lat != nil || lng != nil {
		s.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, s)
}

type shelterPatchInput struct {
	Name             *string   `json:"name"`
	Location         *string   `json:"location"`
	Phone            *string   `json:"phone"`
	Link             *string   `json:"link"`
	Status           *string   `json:"status"`
	Capacity         *int      `json:"capacity"`
	CurrentOccupancy *int      `json:"current_occupancy"`
	AvailableSpaces  *int      `json:"available_spaces"`
	Facilities       *[]string `json:"facilities"`
	ContactPerson    *string   `json:"contact_person"`
	Notes            *string   `json:"notes"`
	Coordinates      *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	OpeningHours *string `json:"opening_hours"`
}

func (h *Handler) PatchShelter(c *gin.Context) {
	id := c.Param("id")
	var in shelterPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
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
	if in.Location != nil {
		add("location=", *in.Location)
	}
	if in.Phone != nil {
		add("phone=", *in.Phone)
	}
	if in.Link != nil {
		add("link=", *in.Link)
	}
	if in.Status != nil {
		add("status=", *in.Status)
	}
	if in.Capacity != nil {
		add("capacity=", *in.Capacity)
	}
	if in.CurrentOccupancy != nil {
		add("current_occupancy=", *in.CurrentOccupancy)
	}
	if in.AvailableSpaces != nil {
		add("available_spaces=", *in.AvailableSpaces)
	}
	if in.Facilities != nil {
		add("facilities=", *in.Facilities)
	}
	if in.ContactPerson != nil {
		add("contact_person=", *in.ContactPerson)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.Coordinates != nil {
		if b, err := json.Marshal(in.Coordinates); err == nil {
			// coordinates is jsonb
			setParts = append(setParts, "coordinates=$"+strconv.Itoa(idx)+"::jsonb")
			args = append(args, string(b))
			idx++
		}
	}
	if in.OpeningHours != nil {
		add("opening_hours=", *in.OpeningHours)
	}
	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}
	// always update updated_at
	setParts = append(setParts, "updated_at=now()")
	query := "update shelters set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,name,location,phone,link,status,capacity,current_occupancy,available_spaces,facilities,contact_person,notes,(coordinates->>'lat')::double precision as lat,(coordinates->>'lng')::double precision as lng,opening_hours,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
	args = append(args, id)
	row := h.pool.QueryRow(ctx, query, args...)
	var s models.Shelter
	var link, contactPerson, notes, opening *string
	var capacity, currentOcc, avail *int
	var facilities []string
	var lat, lng *float64
	var created, updated int64
	if err := row.Scan(&s.ID, &s.Name, &s.Location, &s.Phone, &link, &s.Status, &capacity, &currentOcc, &avail, &facilities, &contactPerson, &notes, &lat, &lng, &opening, &created, &updated); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.Link = link
	s.ContactPerson = contactPerson
	s.Notes = notes
	s.OpeningHours = opening
	s.Capacity = capacity
	s.CurrentOccupancy = currentOcc
	s.AvailableSpaces = avail
	s.Facilities = facilities
	s.CreatedAt = created
	s.UpdatedAt = updated
	if lat != nil || lng != nil {
		s.Coordinates = &struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		}{Lat: lat, Lng: lng}
	}
	c.JSON(http.StatusOK, s)
}
