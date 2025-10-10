package handlers

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"
    "strings"

    "guangfu250923/internal/models"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

type placeCreateInput struct {
    Name               string    `json:"name" binding:"required"`
    Address            string    `json:"address" binding:"required"`
    AddressDescription *string   `json:"address_description"`
    Coordinates        map[string]interface{} `json:"coordinates" binding:"required"`
    Type         string    `json:"type" binding:"required"`
    SubType      *string   `json:"sub_type"`
    InfoSources  []string  `json:"info_sources"`
    VerifiedAt   *int64    `json:"verified_at"`
    WebsiteURL   *string   `json:"website_url"`
    Status       string    `json:"status" binding:"required"`
    Resources    []map[string]interface{} `json:"resources"`
    OpenDate     *string   `json:"open_date"`
    EndDate      *string   `json:"end_date"`
    OpenTime     *string   `json:"open_time"`
    EndTime      *string   `json:"end_time"`
    ContactName  string    `json:"contact_name" binding:"required"`
    ContactPhone string    `json:"contact_phone" binding:"required"`
    Notes        *string   `json:"notes"`
    Tags         []map[string]interface{} `json:"tags"`
    AdditionalInfo map[string]interface{} `json:"additional_info"`
}

func (h *Handler) CreatePlace(c *gin.Context) {
    var in placeCreateInput
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // Status/type validation is enforced by DB constraint; we can do light checks here if desired.
    var coordsJSON *string
    if b, err := json.Marshal(in.Coordinates); err == nil {
        s := string(b)
        coordsJSON = &s
    }
    var resourcesJSON, tagsJSON, addInfoJSON *string
    if in.Resources != nil {
        if b, err := json.Marshal(in.Resources); err == nil {
            s := string(b); resourcesJSON = &s
        }
    }
    if in.Tags != nil {
        if b, err := json.Marshal(in.Tags); err == nil {
            s := string(b); tagsJSON = &s
        }
    }
    if in.AdditionalInfo != nil {
        if b, err := json.Marshal(in.AdditionalInfo); err == nil {
            s := string(b); addInfoJSON = &s
        }
    }
    newID, _ := uuid.NewV7()
    id := newID.String()
    ctx := context.Background()
    var created, updated int64
    err := h.pool.QueryRow(ctx, `insert into places(
        id,name,address,address_description,coordinates,type,sub_type,info_sources,verified_at,website_url,status,resources,open_date,end_date,open_time,end_time,contact_name,contact_phone,notes,tags,additional_info
    ) values($1,$2,$3,$4,$5::jsonb,$6,$7,$8::text[],$9,$10,$11,$12::jsonb,$13,$14,$15,$16,$17,$18,$19,$20::jsonb,$21::jsonb)
    returning extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint`,
        id, in.Name, in.Address, in.AddressDescription, coordsJSON, in.Type, in.SubType, in.InfoSources, in.VerifiedAt, in.WebsiteURL, in.Status, resourcesJSON, in.OpenDate, in.EndDate, in.OpenTime, in.EndTime, in.ContactName, in.ContactPhone, in.Notes, tagsJSON, addInfoJSON,
    ).Scan(&created, &updated)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    out := models.Place{
        ID: id, Name: in.Name, Address: in.Address, AddressDescription: in.AddressDescription, Type: in.Type,
        SubType: in.SubType, InfoSources: in.InfoSources, VerifiedAt: in.VerifiedAt, WebsiteURL: in.WebsiteURL,
        Status: in.Status, OpenDate: in.OpenDate, EndDate: in.EndDate, OpenTime: in.OpenTime, EndTime: in.EndTime,
        ContactName: in.ContactName, ContactPhone: in.ContactPhone, Notes: in.Notes, CreatedAt: created, UpdatedAt: updated,
    }
    out.Coordinates = in.Coordinates
    out.Resources = in.Resources
    out.Tags = in.Tags
    out.AdditionalInfo = in.AdditionalInfo
    c.JSON(http.StatusCreated, out)
}

func (h *Handler) GetPlace(c *gin.Context) {
    id := c.Param("id")
    ctx := context.Background()
    row := h.pool.QueryRow(ctx, `select id,name,address,address_description,coordinates,
        type,sub_type,info_sources,verified_at,website_url,status,resources,tags,additional_info,open_date,end_date,open_time,end_time,contact_name,contact_phone,
        extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from places where id=$1`, id)
    var p models.Place
    var addrDesc, subType, websiteURL, notes *string
    var infoSources []string
    var verifiedAt *int64
    var openDate, endDate, openTime, endTime *string
    var contactName, contactPhone string
    var coordsJSONRaw []byte
    var created, updated int64
    var resourcesJSON, tagsJSON, addInfoJSON []byte
    if err := row.Scan(&p.ID, &p.Name, &p.Address, &addrDesc, &coordsJSONRaw, &p.Type, &subType, &infoSources, &verifiedAt, &websiteURL, &p.Status, &resourcesJSON, &tagsJSON, &addInfoJSON, &openDate, &endDate, &openTime, &endTime, &contactName, &contactPhone, &created, &updated); err != nil {
        if err == pgx.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    p.AddressDescription = addrDesc
    p.SubType = subType
    p.InfoSources = infoSources
    p.VerifiedAt = verifiedAt
    p.WebsiteURL = websiteURL
    p.OpenDate = openDate
    p.EndDate = endDate
    p.OpenTime = openTime
    p.EndTime = endTime
    p.ContactName = contactName
    p.ContactPhone = contactPhone
    p.CreatedAt = created
    p.UpdatedAt = updated
    if len(coordsJSONRaw) > 0 {
        var obj map[string]interface{}
        _ = json.Unmarshal(coordsJSONRaw, &obj)
        p.Coordinates = obj
    }
    if len(resourcesJSON) > 0 {
        var arr []map[string]interface{}
        _ = json.Unmarshal(resourcesJSON, &arr)
        p.Resources = arr
    }
    if len(tagsJSON) > 0 {
        var arr []map[string]interface{}
        _ = json.Unmarshal(tagsJSON, &arr)
        p.Tags = arr
    }
    if len(addInfoJSON) > 0 {
        var m map[string]interface{}
        _ = json.Unmarshal(addInfoJSON, &m)
        p.AdditionalInfo = m
    }
    p.Notes = notes
    c.JSON(http.StatusOK, p)
}

func (h *Handler) ListPlaces(c *gin.Context) {
    limit := parsePositiveInt(c.Query("limit"), 50, 1, 500)
    offset := parsePositiveInt(c.Query("offset"), 0, 0, 1000000)
    status := c.Query("status")
    typ := c.Query("type")
    ctx := context.Background()
    filters := []string{}
    args := []interface{}{}
    if status != "" {
        filters = append(filters, "status=$"+strconv.Itoa(len(args)+1))
        args = append(args, status)
    }
    if typ != "" {
        filters = append(filters, "type=$"+strconv.Itoa(len(args)+1))
        args = append(args, typ)
    }
    countQ := "select count(*) from places"
    dataQ := "select id,name,address,address_description,coordinates, type,sub_type,info_sources,verified_at,website_url,status,resources,tags,additional_info,open_date,end_date,open_time,end_time,contact_name,contact_phone,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint from places"
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
    list := []models.Place{}
    for rows.Next() {
        var p models.Place
        var addrDesc, subType, websiteURL, notes *string
        var infoSources []string
        var verifiedAt *int64
        var openDate, endDate, openTime, endTime *string
        var contactName, contactPhone string
    var coordsJSONRaw []byte
        var created, updated int64
        var resourcesJSON, tagsJSON, addInfoJSON []byte
    if err := rows.Scan(&p.ID, &p.Name, &p.Address, &addrDesc, &coordsJSONRaw, &p.Type, &subType, &infoSources, &verifiedAt, &websiteURL, &p.Status, &resourcesJSON, &tagsJSON, &addInfoJSON, &openDate, &endDate, &openTime, &endTime, &contactName, &contactPhone, &created, &updated); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        p.AddressDescription = addrDesc
        p.SubType = subType
        p.InfoSources = infoSources
        p.VerifiedAt = verifiedAt
        p.WebsiteURL = websiteURL
        p.OpenDate = openDate
        p.EndDate = endDate
        p.OpenTime = openTime
        p.EndTime = endTime
        p.ContactName = contactName
        p.ContactPhone = contactPhone
        p.CreatedAt = created
        p.UpdatedAt = updated
        if len(coordsJSONRaw) > 0 {
            var obj map[string]interface{}
            _ = json.Unmarshal(coordsJSONRaw, &obj)
            p.Coordinates = obj
        }
        if len(resourcesJSON) > 0 {
            var arr []map[string]interface{}
            _ = json.Unmarshal(resourcesJSON, &arr)
            p.Resources = arr
        }
        if len(tagsJSON) > 0 {
            var arr []map[string]interface{}
            _ = json.Unmarshal(tagsJSON, &arr)
            p.Tags = arr
        }
        if len(addInfoJSON) > 0 {
            var m map[string]interface{}
            _ = json.Unmarshal(addInfoJSON, &m)
            p.AdditionalInfo = m
        }
        p.Notes = notes
        list = append(list, p)
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

type placePatchInput struct {
    Name               *string   `json:"name"`
    Address            *string   `json:"address"`
    AddressDescription *string   `json:"address_description"`
    Coordinates        *map[string]interface{} `json:"coordinates"`
    Type         *string  `json:"type"`
    SubType      *string  `json:"sub_type"`
    InfoSources  *[]string `json:"info_sources"`
    VerifiedAt   *int64   `json:"verified_at"`
    WebsiteURL   *string  `json:"website_url"`
    Status       *string  `json:"status"`
    Resources    *[]map[string]interface{} `json:"resources"`
    OpenDate     *string  `json:"open_date"`
    EndDate      *string  `json:"end_date"`
    OpenTime     *string  `json:"open_time"`
    EndTime      *string  `json:"end_time"`
    ContactName  *string  `json:"contact_name"`
    ContactPhone *string  `json:"contact_phone"`
    Notes        *string  `json:"notes"`
    Tags         *[]map[string]interface{} `json:"tags"`
    AdditionalInfo *map[string]interface{} `json:"additional_info"`
}

func (h *Handler) PatchPlace(c *gin.Context) {
    id := c.Param("id")
    var in placePatchInput
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    ctx := context.Background()
    setParts := []string{}
    args := []interface{}{}
    idx := 1
    add := func(expr string, val interface{}) { setParts = append(setParts, expr+"$"+strconv.Itoa(idx)); args = append(args, val); idx++ }
    if in.Name != nil { add("name=", *in.Name) }
    if in.Address != nil { add("address=", *in.Address) }
    if in.AddressDescription != nil { add("address_description=", *in.AddressDescription) }
    if in.Coordinates != nil {
        if b, err := json.Marshal(in.Coordinates); err == nil { setParts = append(setParts, "coordinates=$"+strconv.Itoa(idx)+"::jsonb"); args = append(args, string(b)); idx++ }
    }
    if in.Type != nil { add("type=", *in.Type) }
    if in.SubType != nil { add("sub_type=", *in.SubType) }
    if in.InfoSources != nil { setParts = append(setParts, "info_sources=$"+strconv.Itoa(idx)+"::text[]"); args = append(args, *in.InfoSources); idx++ }
    if in.VerifiedAt != nil { add("verified_at=", *in.VerifiedAt) }
    if in.WebsiteURL != nil { add("website_url=", *in.WebsiteURL) }
    if in.Status != nil { add("status=", *in.Status) }
    if in.Resources != nil { if b, err := json.Marshal(in.Resources); err == nil { setParts = append(setParts, "resources=$"+strconv.Itoa(idx)+"::jsonb"); args = append(args, string(b)); idx++ } }
    if in.OpenDate != nil { add("open_date=", *in.OpenDate) }
    if in.EndDate != nil { add("end_date=", *in.EndDate) }
    if in.OpenTime != nil { add("open_time=", *in.OpenTime) }
    if in.EndTime != nil { add("end_time=", *in.EndTime) }
    if in.ContactName != nil { add("contact_name=", *in.ContactName) }
    if in.ContactPhone != nil { add("contact_phone=", *in.ContactPhone) }
    if in.Notes != nil { add("notes=", *in.Notes) }
    if in.Tags != nil { if b, err := json.Marshal(in.Tags); err == nil { setParts = append(setParts, "tags=$"+strconv.Itoa(idx)+"::jsonb"); args = append(args, string(b)); idx++ } }
    if in.AdditionalInfo != nil { if b, err := json.Marshal(in.AdditionalInfo); err == nil { setParts = append(setParts, "additional_info=$"+strconv.Itoa(idx)+"::jsonb"); args = append(args, string(b)); idx++ } }
    if len(setParts) == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"}); return }
    setParts = append(setParts, "updated_at=now()")
    query := "update places set "+strings.Join(setParts, ",")+" where id=$"+strconv.Itoa(idx)+" returning id,name,address,address_description,coordinates,type,sub_type,info_sources,verified_at,website_url,status,resources,tags,additional_info,open_date,end_date,open_time,end_time,contact_name,contact_phone,extract(epoch from created_at)::bigint,extract(epoch from updated_at)::bigint"
    args = append(args, id)
    row := h.pool.QueryRow(ctx, query, args...)
    var p models.Place
    var addrDesc, subType, websiteURL, notes *string
    var infoSources []string
    var verifiedAt *int64
    var openDate, endDate, openTime, endTime *string
    var contactName, contactPhone string
    var coordsJSONRaw []byte
    var created, updated int64
    var resourcesJSON, tagsJSON, addInfoJSON []byte
    if err := row.Scan(&p.ID, &p.Name, &p.Address, &addrDesc, &coordsJSONRaw, &p.Type, &subType, &infoSources, &verifiedAt, &websiteURL, &p.Status, &resourcesJSON, &tagsJSON, &addInfoJSON, &openDate, &endDate, &openTime, &endTime, &contactName, &contactPhone, &created, &updated); err != nil {
        if err == pgx.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
    }
    p.AddressDescription = addrDesc
    p.SubType = subType
    p.InfoSources = infoSources
    p.VerifiedAt = verifiedAt
    p.WebsiteURL = websiteURL
    p.OpenDate = openDate
    p.EndDate = endDate
    p.OpenTime = openTime
    p.EndTime = endTime
    p.ContactName = contactName
    p.ContactPhone = contactPhone
    p.CreatedAt = created
    p.UpdatedAt = updated
    if len(coordsJSONRaw) > 0 { var obj map[string]interface{}; _ = json.Unmarshal(coordsJSONRaw, &obj); p.Coordinates = obj }
    if len(resourcesJSON) > 0 { var arr []map[string]interface{}; _ = json.Unmarshal(resourcesJSON, &arr); p.Resources = arr }
    if len(tagsJSON) > 0 { var arr []map[string]interface{}; _ = json.Unmarshal(tagsJSON, &arr); p.Tags = arr }
    if len(addInfoJSON) > 0 { var m map[string]interface{}; _ = json.Unmarshal(addInfoJSON, &m); p.AdditionalInfo = m }
    p.Notes = notes
    c.JSON(http.StatusOK, p)
}
