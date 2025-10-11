package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"guangfu250923/internal/models"

	"github.com/gin-gonic/gin"
)

// 對應到 https://github.com/carolchu1208/Hualian-Typhoon-Rescue-Site-Backend-Team/blob/main/spec.md#volunteer_organizations

type createVolunteerOrgInput struct {
	RegistrationStatus string  `json:"registration_status"`
	OrganizationNature string  `json:"organization_nature"`
	OrganizationName   string  `json:"organization_name" binding:"required"`
	Coordinator        string  `json:"coordinator"`
	ContactInfo        string  `json:"contact_info"`
	RegistrationMethod string  `json:"registration_method"`
	ServiceContent     string  `json:"service_content"`
	MeetingInfo        string  `json:"meeting_info"`
	Notes              string  `json:"notes"`
	ImageURL           *string `json:"image_url"`
}

func (h *Handler) CreateVolunteerOrg(c *gin.Context) {
	var in createVolunteerOrgInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := context.Background()
	var id string
	var lastUpdated time.Time
	err := h.pool.QueryRow(ctx, `insert into volunteer_organizations(last_updated,registration_status,organization_nature,organization_name,coordinator,contact_info,registration_method,service_content,meeting_info,notes,image_url) values(now(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10) returning id,last_updated`,
		in.RegistrationStatus, in.OrganizationNature, in.OrganizationName, in.Coordinator, in.ContactInfo, in.RegistrationMethod, in.ServiceContent, in.MeetingInfo, in.Notes, in.ImageURL,
	).Scan(&id, &lastUpdated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := models.VolunteerOrganization{ID: id, LastUpdated: &lastUpdated, RegistrationStatus: in.RegistrationStatus, OrganizationNature: in.OrganizationNature, OrganizationName: in.OrganizationName, Coordinator: in.Coordinator, ContactInfo: in.ContactInfo, RegistrationMethod: in.RegistrationMethod, ServiceContent: in.ServiceContent, MeetingInfo: in.MeetingInfo, Notes: in.Notes, ImageURL: in.ImageURL}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) ListVolunteerOrgs(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 20, 1, 200)
	offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
	ctx := context.Background()
	var total int
	h.pool.QueryRow(ctx, `select count(*) from volunteer_organizations`).Scan(&total)
	rows, err := h.pool.Query(ctx, `select id,last_updated,registration_status,organization_nature,organization_name,coordinator,contact_info,registration_method,service_content,meeting_info,notes,image_url from volunteer_organizations order by last_updated desc limit $1 offset $2`, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	list := []models.VolunteerOrganization{}
	for rows.Next() {
		var vo models.VolunteerOrganization
		if err = rows.Scan(&vo.ID, &vo.LastUpdated, &vo.RegistrationStatus, &vo.OrganizationNature, &vo.OrganizationName, &vo.Coordinator, &vo.ContactInfo, &vo.RegistrationMethod, &vo.ServiceContent, &vo.MeetingInfo, &vo.Notes, &vo.ImageURL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, vo)
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
	c.JSON(http.StatusOK, gin.H{
		"@context":   "https://www.w3.org/ns/hydra/context.jsonld",
		"@type":      "Collection",
		"totalItems": total,
		"member":     list,
		"limit":      limit,
		"offset":     offset,
		"next":       next,
		"previous":   prev,
	})
}

// GetVolunteerOrg returns a single volunteer organization by id
func (h *Handler) GetVolunteerOrg(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, `select id,last_updated,registration_status,organization_nature,organization_name,coordinator,contact_info,registration_method,service_content,meeting_info,notes,image_url from volunteer_organizations where id=$1`, id)
	var vo models.VolunteerOrganization
	if err := row.Scan(&vo.ID, &vo.LastUpdated, &vo.RegistrationStatus, &vo.OrganizationNature, &vo.OrganizationName, &vo.Coordinator, &vo.ContactInfo, &vo.RegistrationMethod, &vo.ServiceContent, &vo.MeetingInfo, &vo.Notes, &vo.ImageURL); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, vo)
}

type patchVolunteerOrgInput struct {
	RegistrationStatus *string `json:"registration_status"`
	OrganizationNature *string `json:"organization_nature"`
	OrganizationName   *string `json:"organization_name"`
	Coordinator        *string `json:"coordinator"`
	ContactInfo        *string `json:"contact_info"`
	RegistrationMethod *string `json:"registration_method"`
	ServiceContent     *string `json:"service_content"`
	MeetingInfo        *string `json:"meeting_info"`
	Notes              *string `json:"notes"`
	ImageURL           *string `json:"image_url"`
}

// PatchVolunteerOrg partially updates a volunteer organization
func (h *Handler) PatchVolunteerOrg(c *gin.Context) {
	id := c.Param("id")
	var in patchVolunteerOrgInput
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
	if in.RegistrationStatus != nil {
		add("registration_status=", *in.RegistrationStatus)
	}
	if in.OrganizationNature != nil {
		add("organization_nature=", *in.OrganizationNature)
	}
	if in.OrganizationName != nil {
		add("organization_name=", *in.OrganizationName)
	}
	if in.Coordinator != nil {
		add("coordinator=", *in.Coordinator)
	}
	if in.ContactInfo != nil {
		add("contact_info=", *in.ContactInfo)
	}
	if in.RegistrationMethod != nil {
		add("registration_method=", *in.RegistrationMethod)
	}
	if in.ServiceContent != nil {
		add("service_content=", *in.ServiceContent)
	}
	if in.MeetingInfo != nil {
		add("meeting_info=", *in.MeetingInfo)
	}
	if in.Notes != nil {
		add("notes=", *in.Notes)
	}
	if in.ImageURL != nil {
		add("image_url=", *in.ImageURL)
	}
	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}
	// always bump last_updated timestamp
	setParts = append(setParts, "last_updated=now()")
	query := "update volunteer_organizations set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,last_updated,registration_status,organization_nature,organization_name,coordinator,contact_info,registration_method,service_content,meeting_info,notes,image_url"
	args = append(args, id)
	ctx := context.Background()
	row := h.pool.QueryRow(ctx, query, args...)
	var vo models.VolunteerOrganization
	if err := row.Scan(&vo.ID, &vo.LastUpdated, &vo.RegistrationStatus, &vo.OrganizationNature, &vo.OrganizationName, &vo.Coordinator, &vo.ContactInfo, &vo.RegistrationMethod, &vo.ServiceContent, &vo.MeetingInfo, &vo.Notes, &vo.ImageURL); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, vo)
}
