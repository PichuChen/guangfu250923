package models

import "time"

// VolunteerOrganization represents volunteer_organizations table.
type VolunteerOrganization struct {
	ID                 string     `json:"id"`
	LastUpdated        *time.Time `json:"last_updated"`
	RegistrationStatus string     `json:"registration_status"`
	OrganizationNature string     `json:"organization_nature"`
	OrganizationName   string     `json:"organization_name"`
	Coordinator        string     `json:"coordinator"`
	ContactInfo        string     `json:"contact_info"`
	RegistrationMethod string     `json:"registration_method"`
	ServiceContent     string     `json:"service_content"`
	MeetingInfo        string     `json:"meeting_info"`
	Notes              string     `json:"notes"`
	ImageURL           *string    `json:"image_url"`
}

// Shelter represents shelters table row
type Shelter struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Location         string   `json:"location"`
	Phone            string   `json:"phone"`
	Link             *string  `json:"link"`
	Status           string   `json:"status"`
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
	CreatedAt    int64   `json:"created_at"`
	UpdatedAt    int64   `json:"updated_at"`
}

// MedicalStation represents medical_stations table row
type MedicalStation struct {
	ID              string   `json:"id"`
	StationType     string   `json:"station_type"`
	Name            string   `json:"name"`
	Location        string   `json:"location"`
	DetailedAddress *string  `json:"detailed_address"`
	Phone           *string  `json:"phone"`
	ContactPerson   *string  `json:"contact_person"`
	Status          string   `json:"status"`
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
	CreatedAt              int64   `json:"created_at"`
	UpdatedAt              int64   `json:"updated_at"`
}

// MentalHealthResource represents mental_health_resources table row
type MentalHealthResource struct {
	ID             string   `json:"id"`
	DurationType   string   `json:"duration_type"`
	Name           string   `json:"name"`
	ServiceFormat  string   `json:"service_format"`
	ServiceHours   string   `json:"service_hours"`
	ContactInfo    string   `json:"contact_info"`
	WebsiteURL     *string  `json:"website_url"`
	TargetAudience []string `json:"target_audience"`
	Specialties    []string `json:"specialties"`
	Languages      []string `json:"languages"`
	IsFree         bool     `json:"is_free"`
	Location       *string  `json:"location"`
	Coordinates    *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	Status           string  `json:"status"`
	Capacity         *int    `json:"capacity"`
	WaitingTime      *string `json:"waiting_time"`
	Notes            *string `json:"notes"`
	EmergencySupport bool    `json:"emergency_support"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
}

// Accommodation represents accommodations table row
type Accommodation struct {
	ID                     string   `json:"id"`
	Township               string   `json:"township"`
	Name                   string   `json:"name"`
	HasVacancy             string   `json:"has_vacancy"`
	AvailablePeriod        string   `json:"available_period"`
	Restrictions           *string  `json:"restrictions"`
	ContactInfo            string   `json:"contact_info"`
	RoomInfo               *string  `json:"room_info"`
	Address                string   `json:"address"`
	Pricing                string   `json:"pricing"`
	InfoSource             *string  `json:"info_source"`
	Notes                  *string  `json:"notes"`
	Capacity               *int     `json:"capacity"`
	Status                 string   `json:"status"`
	RegistrationMethod     *string  `json:"registration_method"`
	Facilities             []string `json:"facilities"`
	DistanceToDisasterArea *string  `json:"distance_to_disaster_area"`
	Coordinates            *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// ShowerStation represents shower_stations table row
type ShowerStation struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Address        string  `json:"address"`
	Phone          *string `json:"phone"`
	FacilityType   string  `json:"facility_type"`
	TimeSlots      string  `json:"time_slots"`
	GenderSchedule *struct {
		Male   []string `json:"male"`
		Female []string `json:"female"`
	} `json:"gender_schedule"`
	AvailablePeriod     string   `json:"available_period"`
	Capacity            *int     `json:"capacity"`
	IsFree              bool     `json:"is_free"`
	Pricing             *string  `json:"pricing"`
	Notes               *string  `json:"notes"`
	InfoSource          *string  `json:"info_source"`
	Status              string   `json:"status"`
	Facilities          []string `json:"facilities"`
	DistanceToGuangfu   *string  `json:"distance_to_guangfu"`
	RequiresAppointment bool     `json:"requires_appointment"`
	ContactMethod       *string  `json:"contact_method"`
	Coordinates         *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// WaterRefillStation represents water_refill_stations table row
type WaterRefillStation struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Address                string   `json:"address"`
	Phone                  *string  `json:"phone"`
	WaterType              string   `json:"water_type"`
	OpeningHours           string   `json:"opening_hours"`
	IsFree                 bool     `json:"is_free"`
	ContainerRequired      *string  `json:"container_required"`
	DailyCapacity          *int     `json:"daily_capacity"`
	Status                 string   `json:"status"`
	WaterQuality           *string  `json:"water_quality"`
	Facilities             []string `json:"facilities"`
	Accessibility          bool     `json:"accessibility"`
	DistanceToDisasterArea *string  `json:"distance_to_disaster_area"`
	Notes                  *string  `json:"notes"`
	InfoSource             *string  `json:"info_source"`
	Coordinates            *struct {
		Lat *float64 `json:"lat"`
		Lng *float64 `json:"lng"`
	} `json:"coordinates"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// Restroom represents restrooms table row
type Restroom struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Address                string   `json:"address"`
	Phone                  *string  `json:"phone"`
	FacilityType           string   `json:"facility_type"`
	OpeningHours           string   `json:"opening_hours"`
	IsFree                 bool     `json:"is_free"`
	MaleUnits              *int     `json:"male_units"`
	FemaleUnits            *int     `json:"female_units"`
	UnisexUnits            *int     `json:"unisex_units"`
	AccessibleUnits        *int     `json:"accessible_units"`
	HasWater               bool     `json:"has_water"`
	HasLighting            bool     `json:"has_lighting"`
	Status                 string   `json:"status"`
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
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// HumanResource represents human_resources view/aggregation row
type HumanResource struct {
	ID                      string   `json:"id"`
	Org                     string   `json:"org"`
	Address                 string   `json:"address"`
	Phone                   *string  `json:"phone"`
	Status                  string   `json:"status"`
	IsCompleted             bool     `json:"is_completed"`
	HasMedical              *bool    `json:"has_medical"`
	PiiDate                 *int64   `json:"pii_date"`
	CreatedAt               int64    `json:"created_at"`
	UpdatedAt               int64    `json:"updated_at"`
	RoleName                string   `json:"role_name"`
	RoleType                string   `json:"role_type"`
	Skills                  []string `json:"skills"`
	Certifications          []string `json:"certifications"`
	ExperienceLevel         *string  `json:"experience_level"`
	LanguageRequirements    []string `json:"language_requirements"`
	HeadcountNeed           int      `json:"headcount_need"`
	HeadcountGot            int      `json:"headcount_got"`
	HeadcountUnit           *string  `json:"headcount_unit"`
	RoleStatus              string   `json:"role_status"`
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

// Supply represents supplies table row
type Supply struct {
	ID        string  `json:"id"`
	Name      *string `json:"name"`
	Address   *string `json:"address"`
	Phone     *string `json:"phone"`
	Notes     *string `json:"notes"`
	PiiDate   *int64  `json:"pii_date"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
}

// SupplyItem represents supply_items table row (corrected naming)
type SupplyItem struct {
	ID            string  `json:"id"`
	SupplyID      string  `json:"supply_id"`
	Tag           *string `json:"tag"`
	Name          *string `json:"name"`
	ReceivedCount int     `json:"recieved_count"`
	TotalCount    int     `json:"total_count"`
	Unit          *string `json:"unit"`
}

// Photo stores metadata for uploaded images, while the actual file lives in R2/S3.
type Photo struct {
	ID               string `json:"id"`
	ObjectKey        string `json:"object_key"`
	OriginalFilename string `json:"original_filename"`
	ContentType      string `json:"content_type"`
	Size             int64  `json:"size"`
	PublicURL        string `json:"public_url"`
	CreatedAt        int64  `json:"created_at"`
}

// SupplyProvider represents supply_providers table row
type SupplyProvider struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	SupplyItemID  string `json:"supply_item_id"`
	Address       string `json:"address"`
	Notes         *string `json:"notes"`
	ProvideCount  int     `json:"provide_count"`
	ProvideUnit   *string `json:"provide_unit"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

// Report represents reports table row
type Report struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	LocationType string  `json:"location_type"`
	Reason       string  `json:"reason"`
	Notes        *string `json:"notes"`
	Status       string  `json:"status"`
	LocationID   string  `json:"location_id"`
	CreatedAt    int64   `json:"created_at"`
	UpdatedAt    int64   `json:"updated_at"`
}

// SpamResult represents spam_result table row
type SpamResult struct {
	ID          string                 `json:"id"`
	TargetID    string                 `json:"target_id"`
	TargetType  string                 `json:"target_type"`
	TargetData  map[string]interface{} `json:"target_data"`
	IsSpam      bool                   `json:"is_spam"`
	Judgment    string                 `json:"judgment"`
	ValidatedAt int64                  `json:"validated_at"`
}

// Place represents places table row
type Place struct {
	ID                string                   `json:"id"`
	Name              string                   `json:"name"`
	Address           string                   `json:"address"`
	AddressDescription *string                 `json:"address_description"`
	Coordinates       map[string]interface{}   `json:"coordinates"`
	Type              string                   `json:"type"`
	SubType           *string                  `json:"sub_type"`
	InfoSources       []string                 `json:"info_sources"`
	VerifiedAt        *int64                   `json:"verified_at"`
	WebsiteURL        *string                  `json:"website_url"`
	Status            string                   `json:"status"`
	Resources         []map[string]interface{} `json:"resources"`
	OpenDate          *string                  `json:"open_date"`
	EndDate           *string                  `json:"end_date"`
	OpenTime          *string                  `json:"open_time"`
	EndTime           *string                  `json:"end_time"`
	ContactName       string                   `json:"contact_name"`
	ContactPhone      string                   `json:"contact_phone"`
	Notes             *string                  `json:"notes"`
	Tags              []map[string]interface{} `json:"tags"`
	AdditionalInfo    map[string]interface{}   `json:"additional_info"`
	CreatedAt         int64                    `json:"created_at"`
	UpdatedAt         int64                    `json:"updated_at"`
}

// RequirementsHR represents requirements_hr table row
type RequirementsHR struct {
	ID            string                   `json:"id"`
	PlaceID       string                   `json:"place_id"`
	RequiredType  string                   `json:"required_type"`
	Name          string                   `json:"name"`
	Unit          string                   `json:"unit"`
	RequireCount  int                      `json:"require_count"`
	ReceivedCount int                      `json:"received_count"`
	Tags          []map[string]interface{} `json:"tags"`
	AdditionalInfo map[string]interface{}  `json:"additional_info"`
	CreatedAt     int64                    `json:"created_at"`
	UpdatedAt     int64                    `json:"updated_at"`
}

// RequirementsSupplies represents requirements_supplies table row
type RequirementsSupplies struct {
	ID            string                   `json:"id"`
	PlaceID       string                   `json:"place_id"`
	RequiredType  string                   `json:"required_type"`
	Name          string                   `json:"name"`
	Unit          string                   `json:"unit"`
	RequireCount  int                      `json:"require_count"`
	ReceivedCount int                      `json:"received_count"`
	Tags          []map[string]interface{} `json:"tags"`
	AdditionalInfo map[string]interface{}  `json:"additional_info"`
	CreatedAt     int64                    `json:"created_at"`
	UpdatedAt     int64                    `json:"updated_at"`
}
