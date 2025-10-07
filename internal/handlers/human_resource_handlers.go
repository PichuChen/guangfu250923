package handlers

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"guangfu250923/internal/models"
)

// ListHumanResources returns paginated human resource rows
func (h *Handler) ListHumanResources(c *gin.Context) {
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	status := c.Query("status")
	roleStatus := c.Query("role_status")
	roleType := c.Query("role_type")

	where := []string{}
	args := []interface{}{}
	idx := 1
	add := func(cond string, val interface{}) {
		where = append(where, cond+"$"+strconv.Itoa(idx))
		args = append(args, val)
		idx++
	}
	if status != "" {
		add("status=", status)
	}
	if roleStatus != "" {
		add("role_status=", roleStatus)
	}
	if roleType != "" {
		add("role_type=", roleType)
	}

	base := `select id,org,address,phone,status,is_completed,has_medical,pii_date,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint,role_name,role_type,coalesce(skills,'{}'),coalesce(certifications,'{}'),experience_level,coalesce(language_requirements,'{}'),headcount_need,headcount_got,headcount_unit,role_status,extract(epoch from shift_start_ts)::bigint,extract(epoch from shift_end_ts)::bigint,shift_notes,extract(epoch from assignment_timestamp)::bigint,assignment_count,assignment_notes,total_roles_in_request,completed_roles_in_request,pending_roles_in_request,total_requests,active_requests,completed_requests,cancelled_requests,total_roles,completed_roles,pending_roles,urgent_requests,medical_requests from human_resources`
	countSQL := `select count(*) from human_resources`
	if len(where) > 0 {
		clause := " where " + join(where, " and ")
		base += clause
		countSQL += clause
	}
	base += " order by created_at desc limit $" + strconv.Itoa(idx) + " offset $" + strconv.Itoa(idx+1)
	args = append(args, limit, offset)

	ctx := context.Background()
	var total int
	if err := h.pool.QueryRow(ctx, countSQL, args[:len(args)-2]...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rows, err := h.pool.Query(ctx, base, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []models.HumanResource{}
	for rows.Next() {
		var hr models.HumanResource
		var skills, certs, langs []string
		var hasMedical *bool
		var expLevel *string
		var headUnit *string
		var shiftStart, shiftEnd, assignmentTs *int64
		var shiftNotes, assignmentNotes *string
		var totalRolesInReq, completedRolesInReq, pendingRolesInReq *int
		var totalReq, activeReq, completedReq, cancelledReq *int
		var totalRoles, completedRoles, pendingRoles *int
		var urgentReq, medicalReq *int
		var piiDate *int64
		if err := rows.Scan(&hr.ID, &hr.Org, &hr.Address, &hr.Phone, &hr.Status, &hr.IsCompleted, &hasMedical, &piiDate, &hr.CreatedAt, &hr.UpdatedAt, &hr.RoleName, &hr.RoleType, &skills, &certs, &expLevel, &langs, &hr.HeadcountNeed, &hr.HeadcountGot, &headUnit, &hr.RoleStatus, &shiftStart, &shiftEnd, &shiftNotes, &assignmentTs, &hr.AssignmentCount, &assignmentNotes, &totalRolesInReq, &completedRolesInReq, &pendingRolesInReq, &totalReq, &activeReq, &completedReq, &cancelledReq, &totalRoles, &completedRoles, &pendingRoles, &urgentReq, &medicalReq); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hr.PiiDate = piiDate
		hr.HasMedical = hasMedical
		hr.Skills = skills
		hr.Certifications = certs
		hr.LanguageRequirements = langs
		hr.ExperienceLevel = expLevel
		hr.HeadcountUnit = headUnit
		hr.ShiftStartTs = shiftStart
		hr.ShiftEndTs = shiftEnd
		hr.ShiftNotes = shiftNotes
		hr.AssignmentTimestamp = assignmentTs
		hr.AssignmentNotes = assignmentNotes
		hr.TotalRolesInRequest = totalRolesInReq
		hr.CompletedRolesInRequest = completedRolesInReq
		hr.PendingRolesInRequest = pendingRolesInReq
		hr.TotalRequests = totalReq
		hr.ActiveRequests = activeReq
		hr.CompletedRequests = completedReq
		hr.CancelledRequests = cancelledReq
		hr.TotalRoles = totalRoles
		hr.CompletedRoles = completedRoles
		hr.PendingRoles = pendingRoles
		hr.UrgentRequests = urgentReq
		hr.MedicalRequests = medicalReq
		list = append(list, hr)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"member":     list,
		"totalItems": total,
		"limit":      limit,
		"offset":     offset,
	})
}

// Helper join (avoid importing strings to keep style consistent)
func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}

// GetHumanResource fetch single by id
func (h *Handler) GetHumanResource(c *gin.Context) {
	id := c.Param("id")
	row := h.pool.QueryRow(context.Background(), `select id,org,address,phone,status,is_completed,has_medical,pii_date,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint,role_name,role_type,coalesce(skills,'{}'),coalesce(certifications,'{}'),experience_level,coalesce(language_requirements,'{}'),headcount_need,headcount_got,headcount_unit,role_status,extract(epoch from shift_start_ts)::bigint,extract(epoch from shift_end_ts)::bigint,shift_notes,extract(epoch from assignment_timestamp)::bigint,assignment_count,assignment_notes,total_roles_in_request,completed_roles_in_request,pending_roles_in_request,total_requests,active_requests,completed_requests,cancelled_requests,total_roles,completed_roles,pending_roles,urgent_requests,medical_requests from human_resources where id=$1`, id)
	var hr models.HumanResource
	var skills, certs, langs []string
	var hasMedical *bool
	var expLevel *string
	var headUnit *string
	var shiftStart, shiftEnd, assignmentTs *int64
	var shiftNotes, assignmentNotes *string
	var totalRolesInReq, completedRolesInReq, pendingRolesInReq *int
	var totalReq, activeReq, completedReq, cancelledReq *int
	var totalRoles, completedRoles, pendingRoles *int
	var urgentReq, medicalReq *int
	var piiDate *int64
	if err := row.Scan(&hr.ID, &hr.Org, &hr.Address, &hr.Phone, &hr.Status, &hr.IsCompleted, &hasMedical, &piiDate, &hr.CreatedAt, &hr.UpdatedAt, &hr.RoleName, &hr.RoleType, &skills, &certs, &expLevel, &langs, &hr.HeadcountNeed, &hr.HeadcountGot, &headUnit, &hr.RoleStatus, &shiftStart, &shiftEnd, &shiftNotes, &assignmentTs, &hr.AssignmentCount, &assignmentNotes, &totalRolesInReq, &completedRolesInReq, &pendingRolesInReq, &totalReq, &activeReq, &completedReq, &cancelledReq, &totalRoles, &completedRoles, &pendingRoles, &urgentReq, &medicalReq); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hr.HasMedical = hasMedical
	hr.PiiDate = piiDate
	hr.Skills = skills
	hr.Certifications = certs
	hr.LanguageRequirements = langs
	hr.ExperienceLevel = expLevel
	hr.HeadcountUnit = headUnit
	hr.ShiftStartTs = shiftStart
	hr.ShiftEndTs = shiftEnd
	hr.ShiftNotes = shiftNotes
	hr.AssignmentTimestamp = assignmentTs
	hr.AssignmentNotes = assignmentNotes
	hr.TotalRolesInRequest = totalRolesInReq
	hr.CompletedRolesInRequest = completedRolesInReq
	hr.PendingRolesInRequest = pendingRolesInReq
	hr.TotalRequests = totalReq
	hr.ActiveRequests = activeReq
	hr.CompletedRequests = completedReq
	hr.CancelledRequests = cancelledReq
	hr.TotalRoles = totalRoles
	hr.CompletedRoles = completedRoles
	hr.PendingRoles = pendingRoles
	hr.UrgentRequests = urgentReq
	hr.MedicalRequests = medicalReq
	c.JSON(http.StatusOK, hr)
}

// ----- Create -----

type humanResourceCreateInput struct {
	Org                  string   `json:"org"`
	Address              string   `json:"address"`
	Phone                string   `json:"phone"`
	Status               string   `json:"status"`
	IsCompleted          bool     `json:"is_completed"`
	HasMedical           *bool    `json:"has_medical"`
	PiiDate              *int64   `json:"pii_date"`
	ValidPin             *string  `json:"valid_pin"`
	RoleName             string   `json:"role_name"`
	RoleType             string   `json:"role_type"`
	Skills               []string `json:"skills"`
	Certifications       []string `json:"certifications"`
	ExperienceLevel      *string  `json:"experience_level"`
	LanguageRequirements []string `json:"language_requirements"`
	HeadcountNeed        int      `json:"headcount_need"`
	HeadcountGot         int      `json:"headcount_got"`
	HeadcountUnit        *string  `json:"headcount_unit"`
	RoleStatus           string   `json:"role_status"`
	ShiftStartTs         *int64   `json:"shift_start_ts"`
	ShiftEndTs           *int64   `json:"shift_end_ts"`
	ShiftNotes           *string  `json:"shift_notes"`
	AssignmentTimestamp  *int64   `json:"assignment_timestamp"`
	AssignmentCount      *int     `json:"assignment_count"`
	AssignmentNotes      *string  `json:"assignment_notes"`
	// Aggregation / derived fields (optional on create)
	TotalRolesInRequest     *int `json:"total_roles_in_request"`
	CompletedRolesInRequest *int `json:"completed_roles_in_request"`
	PendingRolesInRequest   *int `json:"pending_roles_in_request"`
	TotalRequests           *int `json:"total_requests"`
	ActiveRequests          *int `json:"active_requests"`
	CompletedRequests       *int `json:"completed_requests"`
	CancelledRequests       *int `json:"cancelled_requests"`
	TotalRoles              *int `json:"total_roles"`
	CompletedRoles          *int `json:"completed_roles"`
	PendingRoles            *int `json:"pending_roles"`
	UrgentRequests          *int `json:"urgent_requests"`
	MedicalRequests         *int `json:"medical_requests"`
}

func (h *Handler) CreateHumanResource(c *gin.Context) {
	var in humanResourceCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Basic required validation
	// phone 不再必填，移除必填檢查；若未提供將以空字串寫入 (DB 目前允許非空/空字串)
	requiredStr := map[string]string{"org": in.Org, "address": in.Address, "status": in.Status, "role_name": in.RoleName, "role_type": in.RoleType, "role_status": in.RoleStatus}
	for k, v := range requiredStr {
		if strings.TrimSpace(v) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": k + " is required"})
			return
		}
	}
	// valid_pin: if empty or missing, generate one for backward compatibility; otherwise validate format
	if in.ValidPin == nil || strings.TrimSpace(*in.ValidPin) == "" {
		tmp := GeneratePin(6)
		in.ValidPin = &tmp
	} else if !isValidPin6(in.ValidPin) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid_pin must be 6 digits, with 1 - 9"})
		return
	}
	if in.HeadcountNeed <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "headcount_need must be > 0"})
		return
	}
	if in.HeadcountGot < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "headcount_got must be >= 0"})
		return
	}

	newUUID, err := uuid.NewV7()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate UUID: " + err.Error()})
		return
	}
	id := "hr-" + newUUID.String()
	// Convert epoch timestamps to *time.Time
	toTime := func(ts *int64) *time.Time {
		if ts == nil {
			return nil
		}
		t := time.Unix(*ts, 0).UTC()
		return &t
	}
	shiftStart := toTime(in.ShiftStartTs)
	shiftEnd := toTime(in.ShiftEndTs)
	assignmentTs := toTime(in.AssignmentTimestamp)

	// NOTE: keep column count in sync with values placeholders. If you add/remove a column update both lists.
	sql := `insert into human_resources (
			id,org,address,phone,status,is_completed,has_medical,pii_date,role_name,role_type,skills,certifications,experience_level,language_requirements,headcount_need,headcount_got,headcount_unit,role_status,shift_start_ts,shift_end_ts,shift_notes,assignment_timestamp,assignment_count,assignment_notes,total_roles_in_request,completed_roles_in_request,pending_roles_in_request,total_requests,active_requests,completed_requests,cancelled_requests,total_roles,completed_roles,pending_roles,urgent_requests,medical_requests,valid_pin
		) values (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37
		) returning id,org,address,phone,status,is_completed,has_medical,pii_date,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint,role_name,role_type,coalesce(skills,'{}'),coalesce(certifications,'{}'),experience_level,coalesce(language_requirements,'{}'),headcount_need,headcount_got,headcount_unit,role_status,extract(epoch from shift_start_ts)::bigint,extract(epoch from shift_end_ts)::bigint,shift_notes,extract(epoch from assignment_timestamp)::bigint,assignment_count,assignment_notes,total_roles_in_request,completed_roles_in_request,pending_roles_in_request,total_requests,active_requests,completed_requests,cancelled_requests,total_roles,completed_roles,pending_roles,urgent_requests,medical_requests`

	row := h.pool.QueryRow(context.Background(), sql,
		id, in.Org, in.Address, in.Phone, in.Status, in.IsCompleted, in.HasMedical, in.PiiDate, in.RoleName, in.RoleType,
		sliceOrNil(in.Skills), sliceOrNil(in.Certifications), in.ExperienceLevel, sliceOrNil(in.LanguageRequirements),
		in.HeadcountNeed, in.HeadcountGot, in.HeadcountUnit, in.RoleStatus,
		shiftStart, shiftEnd, in.ShiftNotes, assignmentTs, in.AssignmentCount, in.AssignmentNotes,
		in.TotalRolesInRequest, in.CompletedRolesInRequest, in.PendingRolesInRequest, in.TotalRequests, in.ActiveRequests,
		in.CompletedRequests, in.CancelledRequests, in.TotalRoles, in.CompletedRoles, in.PendingRoles, in.UrgentRequests, in.MedicalRequests, in.ValidPin,
	)

	var hr models.HumanResource
	var skills, certs, langs []string
	var hasMedical *bool
	var expLevel *string
	var headUnit *string
	var shiftStartTs, shiftEndTs, assignmentTimestamp *int64
	var shiftNotes, assignmentNotes *string
	var totalRolesInReq, completedRolesInReq, pendingRolesInReq *int
	var totalReq, activeReq, completedReq, cancelledReq *int
	var totalRoles, completedRoles, pendingRoles *int
	var urgentReq, medicalReq *int
	var piiDate2 *int64
	if err := row.Scan(&hr.ID, &hr.Org, &hr.Address, &hr.Phone, &hr.Status, &hr.IsCompleted, &hasMedical, &piiDate2, &hr.CreatedAt, &hr.UpdatedAt, &hr.RoleName, &hr.RoleType, &skills, &certs, &expLevel, &langs, &hr.HeadcountNeed, &hr.HeadcountGot, &headUnit, &hr.RoleStatus, &shiftStartTs, &shiftEndTs, &shiftNotes, &assignmentTimestamp, &hr.AssignmentCount, &assignmentNotes, &totalRolesInReq, &completedRolesInReq, &pendingRolesInReq, &totalReq, &activeReq, &completedReq, &cancelledReq, &totalRoles, &completedRoles, &pendingRoles, &urgentReq, &medicalReq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hr.HasMedical = hasMedical
	hr.PiiDate = piiDate2
	hr.Skills = skills
	hr.Certifications = certs
	hr.LanguageRequirements = langs
	hr.ExperienceLevel = expLevel
	hr.HeadcountUnit = headUnit
	hr.ShiftStartTs = shiftStartTs
	hr.ShiftEndTs = shiftEndTs
	hr.ShiftNotes = shiftNotes
	hr.AssignmentTimestamp = assignmentTimestamp
	hr.AssignmentNotes = assignmentNotes
	hr.TotalRolesInRequest = totalRolesInReq
	hr.CompletedRolesInRequest = completedRolesInReq
	hr.PendingRolesInRequest = pendingRolesInReq
	hr.TotalRequests = totalReq
	hr.ActiveRequests = activeReq
	hr.CompletedRequests = completedReq
	hr.CancelledRequests = cancelledReq
	hr.TotalRoles = totalRoles
	hr.CompletedRoles = completedRoles
	hr.PendingRoles = pendingRoles
	hr.UrgentRequests = urgentReq
	hr.MedicalRequests = medicalReq

	c.JSON(http.StatusCreated, hr)
}

// ----- Patch -----

type humanResourcePatchInput struct {
	ValidPin                *string  `json:"valid_pin"`
	Org                     *string  `json:"org"`
	Address                 *string  `json:"address"`
	Phone                   *string  `json:"phone"`
	Status                  *string  `json:"status"`
	IsCompleted             *bool    `json:"is_completed"`
	HasMedical              *bool    `json:"has_medical"`
	PiiDate                 *int64   `json:"pii_date"`
	RoleName                *string  `json:"role_name"`
	RoleType                *string  `json:"role_type"`
	Skills                  []string `json:"skills"`
	Certifications          []string `json:"certifications"`
	ExperienceLevel         *string  `json:"experience_level"`
	LanguageRequirements    []string `json:"language_requirements"`
	HeadcountNeed           *int     `json:"headcount_need"`
	HeadcountGot            *int     `json:"headcount_got"`
	HeadcountUnit           *string  `json:"headcount_unit"`
	RoleStatus              *string  `json:"role_status"`
	ShiftStartTs            *int64   `json:"shift_start_ts"`
	ShiftEndTs              *int64   `json:"shift_end_ts"`
	ShiftNotes              *string  `json:"shift_notes"`
	AssignmentTimestamp     *int64   `json:"assignment_timestamp"`
	AssignmentCount         *int     `json:"assignment_count"`
	AssignmentNotes         *string  `json:"assignment_notes"`
	TotalRolesInRequest     *int     `json:"total_roles_in_request"`
	CompletedRolesInRequest *int     `json:"completed_roles_in_request"`
	PendingRolesInRequest   *int     `json:"pending_roles_in_request"`
	TotalRequests           *int     `json:"total_requests"`
	ActiveRequests          *int     `json:"active_requests"`
	CompletedRequests       *int     `json:"completed_requests"`
	CancelledRequests       *int     `json:"cancelled_requests"`
	TotalRoles              *int     `json:"total_roles"`
	CompletedRoles          *int     `json:"completed_roles"`
	PendingRoles            *int     `json:"pending_roles"`
	UrgentRequests          *int     `json:"urgent_requests"`
	MedicalRequests         *int     `json:"medical_requests"`
}

func (h *Handler) PatchHumanResource(c *gin.Context) {
	id := c.Param("id")
	var in humanResourcePatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Optional verification (controlled by VERIFY_HR_PIN)
	if os.Getenv("VERIFY_HR_PIN") == "true" {
		// Fetch stored pin (if any)
		var storedPin *string
		if err := h.pool.QueryRow(context.Background(), `select valid_pin from human_resources where id=$1`, id).Scan(&storedPin); err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// PIN behavior:
		// - If VERIFY_HR_PIN=false, do not enforce matching; still generate/set one if missing to onboard records.
		// - If VERIFY_HR_PIN=true, enforce match once a pin exists; else generate/set as needed.
		if storedPin == nil || strings.TrimSpace(*storedPin) == "" {
			// bypass
		} else {
			// PIN exists already in record
			// Determine if this PATCH only updates the allowed trio of fields:
			// status, is_completed, headcount_got. If so, we'll bypass PIN verification.
			if !isOnlyUpdateStatusIsCompletedHeadcountGot(in) {
				// Must provide and match
				if !isValidPin6(in.ValidPin) || *in.ValidPin != *storedPin {
					c.JSON(http.StatusForbidden, gin.H{"error": "invalid pin"})
					return
				}
			}
		}
	}
	setParts := []string{}
	args := []interface{}{}
	idx := 1
	add := func(expr string, v interface{}) {
		setParts = append(setParts, expr+"$"+strconv.Itoa(idx))
		args = append(args, v)
		idx++
	}
	// Simple fields
	if in.Org != nil {
		add("org=", *in.Org)
	}
	if in.Address != nil {
		add("address=", *in.Address)
	}
	if in.Phone != nil {
		add("phone=", *in.Phone)
	}
	if in.Status != nil {
		add("status=", *in.Status)
	}
	if in.IsCompleted != nil {
		add("is_completed=", *in.IsCompleted)
	}
	if in.HasMedical != nil {
		add("has_medical=", *in.HasMedical)
	}
	if in.PiiDate != nil {
		add("pii_date=", *in.PiiDate)
	}
	if in.RoleName != nil {
		add("role_name=", *in.RoleName)
	}
	if in.RoleType != nil {
		add("role_type=", *in.RoleType)
	}
	if in.Skills != nil {
		add("skills=", sliceOrNil(in.Skills))
	}
	if in.Certifications != nil {
		add("certifications=", sliceOrNil(in.Certifications))
	}
	if in.ExperienceLevel != nil {
		add("experience_level=", *in.ExperienceLevel)
	}
	if in.LanguageRequirements != nil {
		add("language_requirements=", sliceOrNil(in.LanguageRequirements))
	}
	if in.HeadcountNeed != nil {
		add("headcount_need=", *in.HeadcountNeed)
	}
	if in.HeadcountGot != nil {
		add("headcount_got=", *in.HeadcountGot)
	}
	if in.HeadcountUnit != nil {
		add("headcount_unit=", *in.HeadcountUnit)
	}
	if in.RoleStatus != nil {
		add("role_status=", *in.RoleStatus)
	}
	// Time fields (convert epoch to timestamptz)
	toTime := func(ts *int64) *time.Time {
		if ts == nil {
			return nil
		}
		t := time.Unix(*ts, 0).UTC()
		return &t
	}
	if in.ShiftStartTs != nil {
		add("shift_start_ts=", toTime(in.ShiftStartTs))
	}
	if in.ShiftEndTs != nil {
		add("shift_end_ts=", toTime(in.ShiftEndTs))
	}
	if in.ShiftNotes != nil {
		add("shift_notes=", *in.ShiftNotes)
	}
	if in.AssignmentTimestamp != nil {
		add("assignment_timestamp=", toTime(in.AssignmentTimestamp))
	}
	if in.AssignmentCount != nil {
		add("assignment_count=", *in.AssignmentCount)
	}
	if in.AssignmentNotes != nil {
		add("assignment_notes=", *in.AssignmentNotes)
	}
	if in.TotalRolesInRequest != nil {
		add("total_roles_in_request=", *in.TotalRolesInRequest)
	}
	if in.CompletedRolesInRequest != nil {
		add("completed_roles_in_request=", *in.CompletedRolesInRequest)
	}
	if in.PendingRolesInRequest != nil {
		add("pending_roles_in_request=", *in.PendingRolesInRequest)
	}
	if in.TotalRequests != nil {
		add("total_requests=", *in.TotalRequests)
	}
	if in.ActiveRequests != nil {
		add("active_requests=", *in.ActiveRequests)
	}
	if in.CompletedRequests != nil {
		add("completed_requests=", *in.CompletedRequests)
	}
	if in.CancelledRequests != nil {
		add("cancelled_requests=", *in.CancelledRequests)
	}
	if in.TotalRoles != nil {
		add("total_roles=", *in.TotalRoles)
	}
	if in.CompletedRoles != nil {
		add("completed_roles=", *in.CompletedRoles)
	}
	if in.PendingRoles != nil {
		add("pending_roles=", *in.PendingRoles)
	}
	if in.UrgentRequests != nil {
		add("urgent_requests=", *in.UrgentRequests)
	}
	if in.MedicalRequests != nil {
		add("medical_requests=", *in.MedicalRequests)
	}
	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}
	setParts = append(setParts, "updated_at=now()")
	query := "update human_resources set " + strings.Join(setParts, ",") + " where id=$" + strconv.Itoa(idx) + " returning id,org,address,phone,status,is_completed,has_medical,pii_date,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint,role_name,role_type,coalesce(skills,'{}'),coalesce(certifications,'{}'),experience_level,coalesce(language_requirements,'{}'),headcount_need,headcount_got,headcount_unit,role_status,extract(epoch from shift_start_ts)::bigint,extract(epoch from shift_end_ts)::bigint,shift_notes,extract(epoch from assignment_timestamp)::bigint,assignment_count,assignment_notes,total_roles_in_request,completed_roles_in_request,pending_roles_in_request,total_requests,active_requests,completed_requests,cancelled_requests,total_roles,completed_roles,pending_roles,urgent_requests,medical_requests"
	args = append(args, id)
	row := h.pool.QueryRow(context.Background(), query, args...)

	var hr models.HumanResource
	var skills, certs, langs []string
	var hasMedical *bool
	var expLevel *string
	var headUnit *string
	var shiftStartTs, shiftEndTs, assignmentTimestamp *int64
	var shiftNotes, assignmentNotes *string
	var totalRolesInReq, completedRolesInReq, pendingRolesInReq *int
	var totalReq, activeReq, completedReq, cancelledReq *int
	var totalRoles, completedRoles, pendingRoles *int
	var urgentReq, medicalReq *int
	var piiDate3 *int64
	if err := row.Scan(&hr.ID, &hr.Org, &hr.Address, &hr.Phone, &hr.Status, &hr.IsCompleted, &hasMedical, &piiDate3, &hr.CreatedAt, &hr.UpdatedAt, &hr.RoleName, &hr.RoleType, &skills, &certs, &expLevel, &langs, &hr.HeadcountNeed, &hr.HeadcountGot, &headUnit, &hr.RoleStatus, &shiftStartTs, &shiftEndTs, &shiftNotes, &assignmentTimestamp, &hr.AssignmentCount, &assignmentNotes, &totalRolesInReq, &completedRolesInReq, &pendingRolesInReq, &totalReq, &activeReq, &completedReq, &cancelledReq, &totalRoles, &completedRoles, &pendingRoles, &urgentReq, &medicalReq); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hr.HasMedical = hasMedical
	hr.PiiDate = piiDate3
	hr.Skills = skills
	hr.Certifications = certs
	hr.LanguageRequirements = langs
	hr.ExperienceLevel = expLevel
	hr.HeadcountUnit = headUnit
	hr.ShiftStartTs = shiftStartTs
	hr.ShiftEndTs = shiftEndTs
	hr.ShiftNotes = shiftNotes
	hr.AssignmentTimestamp = assignmentTimestamp
	hr.AssignmentNotes = assignmentNotes
	hr.TotalRolesInRequest = totalRolesInReq
	hr.CompletedRolesInRequest = completedRolesInReq
	hr.PendingRolesInRequest = pendingRolesInReq
	hr.TotalRequests = totalReq
	hr.ActiveRequests = activeReq
	hr.CompletedRequests = completedReq
	hr.CancelledRequests = cancelledReq
	hr.TotalRoles = totalRoles
	hr.CompletedRoles = completedRoles
	hr.PendingRoles = pendingRoles
	hr.UrgentRequests = urgentReq
	hr.MedicalRequests = medicalReq
	c.JSON(http.StatusOK, hr)
}

// sliceOrNil ensures empty slice becomes nil (to store NULL vs '{}') when appropriate.
func sliceOrNil(s []string) interface{} {
	if s == nil {
		return nil
	}
	return s
}

func isOnlyUpdateStatusIsCompletedHeadcountGot(in humanResourcePatchInput) bool {
	// Track if any of the allowed fields are actually being updated
	hasAnyAllowed := false
	if in.Status != nil {
		hasAnyAllowed = true
	}
	if in.IsCompleted != nil {
		hasAnyAllowed = true
	}
	if in.HeadcountGot != nil {
		hasAnyAllowed = true
	}

	// If any other updatable field is present in the payload, it's not a limited update
	if in.Org != nil || in.Address != nil || in.Phone != nil || in.HasMedical != nil || in.PiiDate != nil ||
		in.RoleName != nil || in.RoleType != nil || in.Skills != nil || in.Certifications != nil ||
		in.ExperienceLevel != nil || in.LanguageRequirements != nil || in.HeadcountNeed != nil ||
		in.HeadcountUnit != nil || in.RoleStatus != nil || in.ShiftStartTs != nil || in.ShiftEndTs != nil ||
		in.ShiftNotes != nil || in.AssignmentTimestamp != nil || in.AssignmentCount != nil ||
		in.AssignmentNotes != nil || in.TotalRolesInRequest != nil || in.CompletedRolesInRequest != nil ||
		in.PendingRolesInRequest != nil || in.TotalRequests != nil || in.ActiveRequests != nil ||
		in.CompletedRequests != nil || in.CancelledRequests != nil || in.TotalRoles != nil ||
		in.CompletedRoles != nil || in.PendingRoles != nil || in.UrgentRequests != nil ||
		in.MedicalRequests != nil {
		return false
	}
	return hasAnyAllowed
}
