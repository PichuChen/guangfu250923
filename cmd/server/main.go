package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"

	"guangfu250923/internal/config"
	"guangfu250923/internal/db"
	"guangfu250923/internal/handlers"
	"guangfu250923/internal/middleware"
	"guangfu250923/internal/sheetcache"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.Load()
	pool, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	defer pool.Close()

	slog.Info("database connected", "cfg", cfg.DBHost+":"+cfg.DBPort+"/"+cfg.DBName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	r := gin.Default()
	// CORS configuration: allow specified front-end origins
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"*",
			"http://localhost:5173",
			"http://127.0.0.1:5500",
			"http://localhost:5050",
			"http://127.0.0.1:5050",
			"https://pinkowo.github.io/",
			"https://guangfu250923-map.pttapp.cc",
			"https://hualian-typhoon-frontend.zeabur.app",
			"https://hualian-typhoon-frontend-dev.zeabur.app",
			"https://sites.google.com/view/guangfu250923",
			"https://692001998-atari-embeds.googleusercontent.com",
			"https://*-atari-embeds.googleusercontent.com",
			"https://hualien-volunteers-frontend-iota.vercel.app/",
		},
		AllowMethods: []string{"GET", "POST", "PATCH", "OPTIONS"},
		// Add "User-Agent" to satisfy Safari (it sometimes includes it in Access-Control-Request-Headers)
		// You may broaden this further or use "*" if you trust clients and want less friction.
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "User-Agent"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           43200 * time.Second, // 12h
	}))
	// Request logging (after CORS so preflight OPTIONS not fully logged body wise)
	r.Use(middleware.RequestLogger(pool, 0))
	// Cache headers for GET responses
	r.Use(middleware.CacheHeaders(0))
	// Security headers (CSP/etc.)
	r.Use(middleware.SecurityHeaders())
	// IP / Country filter for POST/PATCH (uses Cf-Ipcountry header internally + ip_denylist table)
	r.Use(middleware.IPFilter(pool))
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// Swagger UI with custom configuration
	r.StaticFile("/openapi.yaml", "./openapi.yaml")

	url := ginSwagger.URL("/openapi.yaml")
	defaultHost := ginSwagger.DefaultModelsExpandDepth(-1)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url, defaultHost))

	// Sheet cache
	sheetCache := sheetcache.New(cfg.SheetID, cfg.SheetTab)
	pollCtx, cancelPoll := context.WithCancel(context.Background())
	defer cancelPoll()
	sheetCache.StartPolling(pollCtx, cfg.SheetInterval)
	r.GET("/sheet/snapshot", func(c *gin.Context) { c.JSON(http.StatusOK, sheetCache.Snapshot()) })

	h := handlers.New(pool)
	r.POST("/shelters", h.CreateShelter)
	r.GET("/shelters", h.ListShelters)
	r.GET("/shelters/:id", h.GetShelter)
	r.PATCH("/shelters/:id", h.PatchShelter)
	r.POST("/medical_stations", h.CreateMedicalStation)
	r.GET("/medical_stations", h.ListMedicalStations)
	r.GET("/medical_stations/:id", h.GetMedicalStation)
	r.PATCH("/medical_stations/:id", h.PatchMedicalStation)
	r.POST("/mental_health_resources", h.CreateMentalHealthResource)
	r.GET("/mental_health_resources", h.ListMentalHealthResources)
	r.GET("/mental_health_resources/:id", h.GetMentalHealthResource)
	r.PATCH("/mental_health_resources/:id", h.PatchMentalHealthResource)
	r.POST("/accommodations", h.CreateAccommodation)
	r.GET("/accommodations", h.ListAccommodations)
	r.GET("/accommodations/:id", h.GetAccommodation)
	r.PATCH("/accommodations/:id", h.PatchAccommodation)
	r.POST("/shower_stations", h.CreateShowerStation)
	r.GET("/shower_stations", h.ListShowerStations)
	r.GET("/shower_stations/:id", h.GetShowerStation)
	r.PATCH("/shower_stations/:id", h.PatchShowerStation)

	// Water refill stations
	r.POST("/water_refill_stations", h.CreateWaterRefillStation)
	r.GET("/water_refill_stations", h.ListWaterRefillStations)
	r.GET("/water_refill_stations/:id", h.GetWaterRefillStation)
	r.PATCH("/water_refill_stations/:id", h.PatchWaterRefillStation)
	// Restrooms
	r.POST("/restrooms", h.CreateRestroom)
	r.GET("/restrooms", h.ListRestrooms)
	r.GET("/restrooms/:id", h.GetRestroom)
	r.PATCH("/restrooms/:id", h.PatchRestroom)
	r.POST("/volunteer_organizations", h.CreateVolunteerOrg)
	r.GET("/volunteer_organizations", h.ListVolunteerOrgs)
	r.GET("/volunteer_organizations/:id", h.GetVolunteerOrg)
	r.PATCH("/volunteer_organizations/:id", h.PatchVolunteerOrg)
	// Human resources
	r.GET("/human_resources", h.ListHumanResources)
	r.GET("/human_resources/:id", h.GetHumanResource)
	r.POST("/human_resources", h.CreateHumanResource)
	r.PATCH("/human_resources/:id", h.PatchHumanResource)
	// Supplies (new domain) & supply items (renamed from suppily)
	r.POST("/supplies", h.CreateSupply)
	r.GET("/supplies", h.ListSupplies)
	r.GET("/supplies/:id", h.GetSupply)
	// 2025-10-01 要求先關起來
	// r.PATCH("/supplies/:id", h.PatchSupply)
	r.POST("/supplies/:id", h.DistributeSupplyItems) // 批次配送 (累加 recieved_count)
	r.POST("/supply_items", h.CreateSupplyItem)
	r.GET("/supply_items", h.ListSupplyItems)
	r.GET("/supply_items/:id", h.GetSupplyItem)
	// 2025-10-01 要求先關起來
	// r.PATCH("/supply_items/:id", h.PatchSupplyItem)
	// Admin: request logs
	r.GET("/_admin/request_logs", h.ListRequestLogs)

	// Reports (incidents)
	r.POST("/reports", h.CreateReport)
	r.GET("/reports", h.ListReports)
	r.GET("/reports/:id", h.GetReport)
	r.PATCH("/reports/:id", h.PatchReport)

	// Turnstile test endpoint (POST only): echo JSON payload for frontend debugging
	r.POST("/__test_turnstile", middleware.TurnstileVerifier(), func(c *gin.Context) {
		var payload any
		if b, err := io.ReadAll(c.Request.Body); err == nil {
			_ = json.Unmarshal(b, &payload)
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "payload": payload})
	})

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	log.Printf("server listening on :%s", cfg.Port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
