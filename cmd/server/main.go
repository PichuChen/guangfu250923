package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"guangfu250923/internal/config"
	"guangfu250923/internal/db"
	"guangfu250923/internal/handlers"
	"guangfu250923/internal/middleware"
	"guangfu250923/internal/sheetcache"
	"guangfu250923/internal/storage"

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
			"http://localhost:5173", // 一般開發用
			"http://localhost:3000", // Next.js 一般開發用
			"http://127.0.0.1:5500",
			"http://localhost:5050",
			"http://127.0.0.1:5050",
			"https://pinkowo.github.io",           // pink 開發用
			"https://guangfu250923-map.pttapp.cc", // https://guangfu250923-map.pttapp.cc/map.html
			// "https://sites.google.com/view/guangfu250923", // 從未使用
			// "https://hero-guagfu.github.io", // 不應該使用了
			"https://hualien-volunteers-frontend-iota.vercel.app", // 志工媒合在這邊
			"https://guangfu-hero.pttapp.cc",                      // 要拿掉了
			"https://gf250923.org",                                // 新主站
		},
		AllowMethods: []string{"GET", "POST", "PATCH", "OPTIONS"},
		// Add "User-Agent" to satisfy Safari (it sometimes includes it in Access-Control-Request-Headers)
		// You may broaden this further or use "*" if you trust clients and want less friction.
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "User-Agent", "X-Api-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           43200 * time.Second, // 12h
	}))
	// Request logging (after CORS so preflight OPTIONS not fully logged body wise)
	r.Use(middleware.RequestLogger(pool, 0))
	// In-memory GET cache (simple TTL) — must run before CacheHeaders to serve from memory when possible

	cacheTTL, _ := strconv.Atoi(os.Getenv("MEM_CACHE_TTL_SEC"))
	if cacheTTL <= 0 {
		cacheTTL = 60 // default 60s
	}
	r.Use(middleware.MemoryCache(time.Duration(cacheTTL)*time.Second, 1<<20))
	// Cache invalidator after handlers on writes; we place it early so it runs for all routes
	r.Use(middleware.MemoryCacheInvalidator())
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

	// Setup S3 uploader (optional; if not configured, photo upload will return 503)
	var uploader *storage.S3Uploader
	if cfg.S3Bucket != "" {
		up, err := storage.NewS3Uploader(context.Background(), cfg)
		if err != nil {
			slog.Error("s3 uploader init failed", "err", err)
		} else {
			uploader = up
		}
	}

	h := handlers.New(pool, uploader)
	// LINE Login endpoints
	r.GET("/auth/line/start", h.StartLineAuth)
	r.POST("/auth/line/token", h.ExchangeLineToken)
	r.POST("/shelters", h.CreateShelter)
	r.GET("/shelters", h.ListShelters)
	r.GET("/shelters/:id", h.GetShelter)
	r.DELETE("/shelters/:id", middleware.ModifyAPIKeyRequired(), h.DeleteShelter)
	// 2025-10-06 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/shelters/:id", middleware.ModifyAPIKeyRequired(), h.PatchShelter)
	r.POST("/medical_stations", h.CreateMedicalStation)
	r.GET("/medical_stations", h.ListMedicalStations)
	r.GET("/medical_stations/:id", h.GetMedicalStation)
	r.DELETE("/medical_stations/:id", middleware.ModifyAPIKeyRequired(), h.DeleteMedicalStation)
	// 2025-10-06 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/medical_stations/:id", middleware.ModifyAPIKeyRequired(), h.PatchMedicalStation)
	r.POST("/mental_health_resources", h.CreateMentalHealthResource)
	r.GET("/mental_health_resources", h.ListMentalHealthResources)
	r.GET("/mental_health_resources/:id", h.GetMentalHealthResource)
	r.DELETE("/mental_health_resources/:id", middleware.ModifyAPIKeyRequired(), h.DeleteMentalHealthResource)
	// 2025-10-06 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/mental_health_resources/:id", middleware.ModifyAPIKeyRequired(), h.PatchMentalHealthResource)
	r.POST("/accommodations", h.CreateAccommodation)
	r.GET("/accommodations", h.ListAccommodations)
	r.GET("/accommodations/:id", h.GetAccommodation)
	r.DELETE("/accommodations/:id", middleware.ModifyAPIKeyRequired(), h.DeleteAccommodation)
	// 2025-10-06 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/accommodations/:id", middleware.ModifyAPIKeyRequired(), h.PatchAccommodation)
	r.POST("/shower_stations", h.CreateShowerStation)
	r.GET("/shower_stations", h.ListShowerStations)
	r.GET("/shower_stations/:id", h.GetShowerStation)
	r.DELETE("/shower_stations/:id", middleware.ModifyAPIKeyRequired(), h.DeleteShowerStation)
	// 2025-10-06 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/shower_stations/:id", middleware.ModifyAPIKeyRequired(), h.PatchShowerStation)

	// Water refill stations
	r.POST("/water_refill_stations", h.CreateWaterRefillStation)
	r.GET("/water_refill_stations", h.ListWaterRefillStations)
	r.GET("/water_refill_stations/:id", h.GetWaterRefillStation)
	r.DELETE("/water_refill_stations/:id", middleware.ModifyAPIKeyRequired(), h.DeleteWaterRefillStation)
	// 2025-10-06 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/water_refill_stations/:id", middleware.ModifyAPIKeyRequired(), h.PatchWaterRefillStation)
	// Restrooms
	r.POST("/restrooms", h.CreateRestroom)
	r.GET("/restrooms", h.ListRestrooms)
	r.GET("/restrooms/:id", h.GetRestroom)
	r.DELETE("/restrooms/:id", middleware.ModifyAPIKeyRequired(), h.DeleteRestroom)
	// 2025-10-06 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/restrooms/:id", h.PatchRestroom)
	r.POST("/volunteer_organizations", h.CreateVolunteerOrg)
	r.GET("/volunteer_organizations", h.ListVolunteerOrgs)
	r.GET("/volunteer_organizations/:id", h.GetVolunteerOrg)
	r.DELETE("/volunteer_organizations/:id", middleware.ModifyAPIKeyRequired(), h.DeleteVolunteerOrg)
	// 2025-10-06 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/volunteer_organizations/:id", middleware.ModifyAPIKeyRequired(), h.PatchVolunteerOrg)
	// Human resources
	r.GET("/human_resources", h.ListHumanResources)
	r.GET("/human_resources/:id", h.GetHumanResource)
	r.POST("/human_resources", h.CreateHumanResource)
	r.DELETE("/human_resources/:id", middleware.ModifyAPIKeyRequired(), h.DeleteHumanResource)
	// 2025-10-06 因為需要用這個 api 進行到位人數確認，所以是唯一開放的 PATCH api
	// 2025-10-08 驗證 API Key：在 handler 內部判斷是否僅更新 status/is_completed/headcount_got，若非僅更新這三者才要求 API Key
	r.PATCH("/human_resources/:id", h.PatchHumanResource)
	// Supplies (new domain) & supply items (renamed from suppily)
	r.POST("/supplies", h.CreateSupply)
	r.GET("/supplies", h.ListSupplies)
	r.GET("/supplies/:id", h.GetSupply)
	r.DELETE("/supplies/:id", middleware.ModifyAPIKeyRequired(), h.DeleteSupply)
	// 2025-10-01 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/supplies/:id", middleware.ModifyAPIKeyRequired(), h.PatchSupply)
	r.POST("/supplies/:id", h.DistributeSupplyItems) // 批次配送 (累加 recieved_count)
	r.POST("/supply_items", h.CreateSupplyItem)
	r.GET("/supply_items", h.ListSupplyItems)
	r.GET("/supply_items/:id", h.GetSupplyItem)
	r.DELETE("/supply_items/:id", middleware.ModifyAPIKeyRequired(), h.DeleteSupplyItem)
	// 2025-10-01 要求先關起來
	// 2025-10-08 打開來，但是要求驗證 API Key， 提供第三方進行資料同步
	r.PATCH("/supply_items/:id", middleware.ModifyAPIKeyRequired(), h.PatchSupplyItem)
	// Admin: request logs
	r.GET("/_admin/request_logs", h.ListRequestLogs)

	// Reports (incidents)
	r.POST("/reports", h.CreateReport)
	r.GET("/reports", h.ListReports)
	r.GET("/reports/:id", h.GetReport)
	r.PATCH("/reports/:id", h.PatchReport)

	// Spam detection results
	spamResultAPIKey := os.Getenv("SPAM_RESULT_API_KEY")
	r.POST("/spam_results", middleware.APIKeyVerifier(spamResultAPIKey), h.CreateSpamResult)
	r.GET("/spam_results", h.ListSpamResults)
	r.GET("/spam_results/:id", h.GetSpamResult)
	r.PATCH("/spam_results/:id", middleware.APIKeyVerifier(spamResultAPIKey), h.PatchSpamResult)

	// Supply item providers
	r.POST("/supply_providers", h.CreateSupplyProvider)
	r.GET("/supply_providers", h.ListSupplyProviders)
	r.GET("/supply_providers/:id", h.GetSupplyProvider)
	r.PATCH("/supply_providers/:id", h.PatchSupplyProvider)

	// Places
	r.POST("/places", h.CreatePlace)
	r.GET("/places", h.ListPlaces)
	r.GET("/places/:id", h.GetPlace)
	r.DELETE("/places/:id", middleware.ModifyAPIKeyRequired(), h.DeletePlace)
	r.PATCH("/places/:id", middleware.ModifyAPIKeyRequired(), h.PatchPlace)

	// Requirements HR
	r.POST("/requirements_hr", h.CreateRequirementsHR)
	r.GET("/requirements_hr", h.ListRequirementsHR)
	r.GET("/requirements_hr/:id", h.GetRequirementsHR)
	r.DELETE("/requirements_hr/:id", middleware.ModifyAPIKeyRequired(), h.DeleteRequirementsHR)
	r.PATCH("/requirements_hr/:id", middleware.ModifyAPIKeyRequired(), h.PatchRequirementsHR)

	// Requirements Supplies
	r.POST("/requirements_supplies", h.CreateRequirementsSupplies)
	r.GET("/requirements_supplies", h.ListRequirementsSupplies)
	r.GET("/requirements_supplies/:id", h.GetRequirementsSupplies)
	r.DELETE("/requirements_supplies/:id", middleware.ModifyAPIKeyRequired(), h.DeleteRequirementsSupplies)
	r.PATCH("/requirements_supplies/:id", middleware.ModifyAPIKeyRequired(), h.PatchRequirementsSupplies)

	// Photo upload endpoint for disaster victims (protected by Turnstile if enabled)
	r.POST("/uploads/photos", h.UploadPhoto)
	// Public photo route using uuidv7 id stored in DB (supports ?thumbnail=small|medium|large|original)
	r.GET("/photos/:id", h.GetPhoto)

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
