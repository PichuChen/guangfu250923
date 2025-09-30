package main

import (
	"context"
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
			"http://localhost:5050",
			"http://127.0.0.1:5050",
			"https://sites.google.com/view/guangfu250923",
		},
		AllowMethods:     []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           43200 * time.Second, // 12h
	}))
	r.Use(middleware.RequestLogger(pool, 0))
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
	r.POST("/requests", h.CreateRequest)
	r.GET("/requests", h.ListRequests)
	r.POST("/supplies/distribute", h.DistributeSupplies)
	r.GET("/supplies", h.ListSupplies)
	r.GET("/supplies_overview", h.ListSuppliesOverview)
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
	// Admin: request logs
	r.GET("/_admin/request_logs", h.ListRequestLogs)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
	log.Printf("server listening on :%s", cfg.Port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
