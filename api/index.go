package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	router *gin.Engine
	pipedriveService *PipedriveService
)

func init() {
	// Load environment variables
	config := LoadConfig()

	// Initialize services
	pipedriveService = NewPipedriveService(config)

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router ONCE
	router = gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	setupRoutes()
}

func setupRoutes() {
	// Health check endpoint
	router.GET("/health", HealthCheckHandler)
	router.GET("/api/health", HealthCheckHandler)

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "running",
			"message": "PipCal Webhook Server",
			"version": "2.0",
			"endpoints": gin.H{
				"health": "/health",
				"webhooks": gin.H{
					"retell": "/webhook/retell",
					"cal": "/webhook/cal",
					"retell_analyzed": "/webhook/retell/analyzed",
					"pipedrive_lead": "/webhook/pipedrive/lead",
				},
				"test": gin.H{
					"completed": "/test/completed",
					"pipedrive_lead": "/test/pipedrive-lead",
				},
			},
		})
	})
	router.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "running",
			"message": "PipCal Webhook Server",
			"version": "2.0",
		})
	})

	// Webhook endpoints
	router.POST("/webhook/retell", RetellWebhookHandler(pipedriveService))
	router.POST("/webhook/cal", CalWebhookHandler(pipedriveService))
	router.POST("/webhook/retell/analyzed", RetellCallAnalyzedHandler(pipedriveService))
	router.POST("/webhook/pipedrive/lead", PipedriveLeadWebhookHandler(pipedriveService))

	// API versions
	router.POST("/api/webhook/retell", RetellWebhookHandler(pipedriveService))
	router.POST("/api/webhook/cal", CalWebhookHandler(pipedriveService))
	router.POST("/api/webhook/retell/analyzed", RetellCallAnalyzedHandler(pipedriveService))
	router.POST("/api/webhook/pipedrive/lead", PipedriveLeadWebhookHandler(pipedriveService))

	// Test endpoints
	router.POST("/test/completed", func(c *gin.Context) {
		testData := RetellWebhookPayload{
			Event:        "call.completed",
			CallID:       "test-call-" + strconv.FormatInt(time.Now().Unix(), 10),
			ContactPhone: "+1234567890",
			Duration:     "00:02:30",
			Status:       "completed",
			Transcript:   "This is a test call transcript for completed call.",
			Timestamp:    time.Now().Format(time.RFC3339),
		}

		if err := pipedriveService.ProcessRetellCall(testData); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Test failed: " + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Test completed call webhook sent successfully!",
			"data":    testData,
		})
	})

	router.POST("/test/pipedrive-lead", func(c *gin.Context) {
		testData := PipedriveLeadWebhookPayload{
			Data: struct {
				AddTime            string                 `json:"add_time"`
				Channel            interface{}            `json:"channel"`
				ChannelID          interface{}            `json:"channel_id"`
				CreatorID          int                    `json:"creator_id"`
				CustomFields       map[string]interface{} `json:"custom_fields"`
				ExpectedCloseDate  interface{}            `json:"expected_close_date"`
				ID                 string                 `json:"id"`
				IsArchived         bool                   `json:"is_archived"`
				LabelIDs           []string               `json:"label_ids"`
				NextActivityID     interface{}            `json:"next_activity_id"`
				OrganizationID     interface{}            `json:"organization_id"`
				Origin             string                 `json:"origin"`
				OriginID           interface{}            `json:"origin_id"`
				OwnerID            int                    `json:"owner_id"`
				PersonID           int                    `json:"person_id"`
				SourceName         string                 `json:"source_name"`
				Title              string                 `json:"title"`
				UpdateTime         string                 `json:"update_time"`
				WasSeen            bool                   `json:"was_seen"`
				Value              interface{}            `json:"value"`
			}{
				AddTime:    time.Now().Format(time.RFC3339),
				CreatorID:  23836724,
				ID:         "test-lead-" + strconv.FormatInt(time.Now().Unix(), 10),
				IsArchived: false,
				LabelIDs:   []string{"8a48bd05-c7b3-42d7-824b-298d50409325"},
				Origin:     "ManuallyCreated",
				OwnerID:    23836724,
				PersonID:   139,
				SourceName: "Test Lead",
				Title:      "Test Lead - " + strconv.FormatInt(time.Now().Unix(), 10),
				UpdateTime: time.Now().Format(time.RFC3339),
				WasSeen:    true,
			},
			Meta: struct {
				Action             string   `json:"action"`
				CompanyID          string   `json:"company_id"`
				CorrelationID      string   `json:"correlation_id"`
				EntityID           string   `json:"entity_id"`
				Entity             string   `json:"entity"`
				ID                 string   `json:"id"`
				IsBulkEdit         bool     `json:"is_bulk_edit"`
				Timestamp          string   `json:"timestamp"`
				Type               string   `json:"type"`
				UserID             string   `json:"user_id"`
				Version            string   `json:"version"`
				WebhookID          string   `json:"webhook_id"`
				WebhookOwnerID     string   `json:"webhook_owner_id"`
				ChangeSource       string   `json:"change_source"`
				PermittedUserIDs   []string `json:"permitted_user_ids"`
				Attempt            int      `json:"attempt"`
				Host               string   `json:"host"`
			}{
				Action:        "create",
				CompanyID:     "13923453",
				CorrelationID: "test-correlation-" + strconv.FormatInt(time.Now().Unix(), 10),
				EntityID:      "test-entity-" + strconv.FormatInt(time.Now().Unix(), 10),
				Entity:        "lead",
				ID:            "test-meta-" + strconv.FormatInt(time.Now().Unix(), 10),
				IsBulkEdit:    false,
				Timestamp:     time.Now().Format(time.RFC3339),
				Type:          "general",
				UserID:        "23836724",
				Version:       "2.0",
				WebhookID:     "3046302",
				WebhookOwnerID: "23836724",
				ChangeSource:  "app",
				PermittedUserIDs: []string{"23821159", "23825834", "23827748", "23836724"},
				Attempt:       1,
				Host:          "mybusinessportalcloud.pipedrive.com",
			},
		}

		if err := pipedriveService.ProcessPipedriveLead(testData); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Test failed: " + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Test Pipedrive lead webhook sent successfully!",
			"data":    testData,
		})
	})

	log.Printf("✅ Routes configured")
}

// Handler is the Vercel serverless function entry point
func Handler(w http.ResponseWriter, r *http.Request) {
	// Log the request
	log.Printf("📥 Request: %s %s", r.Method, r.URL.Path)

	// Serve the request using Gin router
	router.ServeHTTP(w, r)
}
