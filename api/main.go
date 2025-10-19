package handler

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Handler is the main entry point for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	// Set Gin to release mode for Vercel
	gin.SetMode(gin.ReleaseMode)
	
	// Create Gin router
	router := gin.New()
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

	// Load configuration
	config := LoadConfig()

	// Initialize services
	pipedriveService := NewPipedriveService(config)

	// Serve static files
	router.Static("/static", "./static")
	router.LoadHTMLGlob("static/*.html")

	// Health check endpoint
	router.GET("/health", HealthCheckHandler)

	// Webhook endpoints
	router.POST("/webhook/retell", RetellWebhookHandler(pipedriveService))
	router.POST("/webhook/cal", CalWebhookHandler(pipedriveService))
	router.POST("/webhook/retell/analyzed", RetellCallAnalyzedHandler(pipedriveService))
	router.POST("/webhook/pipedrive/lead", PipedriveLeadWebhookHandler(pipedriveService))

	// Test endpoints
	router.POST("/test/completed", func(c *gin.Context) {
		testData := RetellWebhookPayload{
			CallID:       "test-completed-" + strconv.FormatInt(time.Now().Unix(), 10),
			ContactPhone: "+1234567890",
			Transcript:   "Hello, this is a test call. I am interested in your services and would like to schedule a follow-up meeting. The pricing looks reasonable.",
			Duration:     "00:03:45",
			Status:       "completed",
			Timestamp:    time.Now().Format(time.RFC3339),
			Event:        "call.completed",
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
			"message": "Test completed call sent successfully!",
			"data":    testData,
		})
	})

	router.POST("/test/hangup", func(c *gin.Context) {
		testData := RetellWebhookPayload{
			CallID:       "test-hangup-" + strconv.FormatInt(time.Now().Unix(), 10),
			ContactPhone: "+1987654321",
			Transcript:   "Hello, I am calling about your services but I need to hang up now. Please call me back later.",
			Duration:     "00:01:30",
			Status:       "hangup",
			Timestamp:    time.Now().Format(time.RFC3339),
			Event:        "call.hangup",
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
			"message": "Test hangup call sent successfully!",
			"data":    testData,
		})
	})

	router.POST("/test/optout", func(c *gin.Context) {
		testData := RetellWebhookPayload{
			CallID:       "test-optout-" + strconv.FormatInt(time.Now().Unix(), 10),
			ContactPhone: "+1555123456",
			Transcript:   "Please remove me from your calling list. I do not want to receive any more calls from your company.",
			Duration:     "00:00:45",
			Status:       "optout",
			Timestamp:    time.Now().Format(time.RFC3339),
			Event:        "call.optout",
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
			"message": "Test optout call sent successfully!",
			"data":    testData,
		})
	})

	router.POST("/test/appointment", func(c *gin.Context) {
		testData := CalWebhookPayload{
			TriggerEvent: "BOOKING_CREATED",
			CreatedAt:    time.Now().Format(time.RFC3339),
			Payload: struct {
				ID        int    `json:"id"`
				Title     string `json:"title"`
				StartTime string `json:"startTime"`
				EndTime   string `json:"endTime"`
				Attendees []struct {
					Email string `json:"email"`
					Name  string `json:"name"`
				} `json:"attendees"`
				Location string `json:"location"`
			}{
				ID:        12345,
				Title:     "Product Demo Meeting",
				StartTime: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				EndTime:   time.Now().Add(25 * time.Hour).Format(time.RFC3339),
				Attendees: []struct {
					Email string `json:"email"`
					Name  string `json:"name"`
				}{
					{Email: "test@example.com", Name: "Test User"},
				},
				Location: "https://cal.com/meeting/test123",
			},
		}

		if err := pipedriveService.ProcessCalAppointment(testData); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Test failed: " + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Test appointment sent successfully!",
			"data":    testData,
		})
	})

	// Root route - serve test page
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "PipCal Webhook Server",
		})
	})

	// Serve the request
	router.ServeHTTP(w, r)
}

func main() {
	// Check if running on Vercel
	if os.Getenv("VERCEL") == "1" {
		// Vercel deployment
		http.HandleFunc("/", Handler)
		log.Println("🚀 Starting PipCal Webhook Server on Vercel")
		log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
	} else {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️  No .env file found, using environment variables")
	} else {
		log.Printf("✅ Loaded configuration from .env file")
	}

	// Load configuration from environment variables
	config := LoadConfig()

	// Set Gin mode based on configuration
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize services with configuration
	pipedriveService := NewPipedriveService(config)

	// Create Gin router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add CORS middleware for webhook testing
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

	// Serve static files
	router.Static("/static", "./static")
	router.LoadHTMLGlob("static/*.html")

	// Health check endpoint
	router.GET("/health", HealthCheckHandler)

	// Webhook endpoints
	router.POST("/webhook/retell", RetellWebhookHandler(pipedriveService))
	router.POST("/webhook/cal", CalWebhookHandler(pipedriveService))
	router.POST("/webhook/retell/analyzed", RetellCallAnalyzedHandler(pipedriveService))
	router.POST("/webhook/pipedrive/lead", PipedriveLeadWebhookHandler(pipedriveService))

	// Test endpoints
	router.POST("/test/completed", func(c *gin.Context) {
		testData := RetellWebhookPayload{
			CallID:       "test-completed-" + strconv.FormatInt(time.Now().Unix(), 10),
			ContactPhone: "+1234567890",
			Transcript:   "Hello, this is a test call. I am interested in your services and would like to schedule a follow-up meeting. The pricing looks reasonable.",
			Duration:     "00:03:45",
			Status:       "completed",
			Timestamp:    time.Now().Format(time.RFC3339),
			Event:        "call.completed",
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
			"message": "Test completed call sent successfully!",
			"data":    testData,
		})
	})

	router.POST("/test/hangup", func(c *gin.Context) {
		testData := RetellWebhookPayload{
			CallID:       "test-hangup-" + strconv.FormatInt(time.Now().Unix(), 10),
			ContactPhone: "+1987654321",
			Transcript:   "Hello, I am calling about your services but I need to hang up now. Please call me back later.",
			Duration:     "00:01:30",
			Status:       "hangup",
			Timestamp:    time.Now().Format(time.RFC3339),
			Event:        "call.hangup",
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
			"message": "Test hangup call sent successfully!",
			"data":    testData,
		})
	})

	router.POST("/test/optout", func(c *gin.Context) {
		testData := RetellWebhookPayload{
			CallID:       "test-optout-" + strconv.FormatInt(time.Now().Unix(), 10),
			ContactPhone: "+1555123456",
			Transcript:   "Please remove me from your calling list. I do not want to receive any more calls from your company.",
			Duration:     "00:00:45",
			Status:       "optout",
			Timestamp:    time.Now().Format(time.RFC3339),
			Event:        "call.optout",
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
			"message": "Test optout call sent successfully!",
			"data":    testData,
		})
	})

	router.POST("/test/appointment", func(c *gin.Context) {
		testData := CalWebhookPayload{
			TriggerEvent: "BOOKING_CREATED",
			CreatedAt:    time.Now().Format(time.RFC3339),
			Payload: struct {
				ID        int    `json:"id"`
				Title     string `json:"title"`
				StartTime string `json:"startTime"`
				EndTime   string `json:"endTime"`
				Attendees []struct {
					Email string `json:"email"`
					Name  string `json:"name"`
				} `json:"attendees"`
				Location string `json:"location"`
			}{
				ID:        12345,
				Title:     "Product Demo Meeting",
				StartTime: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				EndTime:   time.Now().Add(25 * time.Hour).Format(time.RFC3339),
				Attendees: []struct {
					Email string `json:"email"`
					Name  string `json:"name"`
				}{
					{Email: "test@example.com", Name: "Test User"},
				},
				Location: "https://cal.com/meeting/test123",
			},
		}

		if err := pipedriveService.ProcessCalAppointment(testData); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Test failed: " + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Test appointment sent successfully!",
			"data":    testData,
		})
	})

	// Root route - serve test page
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "PipCal Webhook Server",
		})
	})

	log.Printf("🚀 Starting PipCal Webhook Server on %s:%s", config.Host, config.Port)
	log.Printf("📋 Available endpoints:")
	log.Printf("   GET  /")
	log.Printf("   GET  /health")
	log.Printf("   POST /webhook/retell")
	log.Printf("   POST /webhook/cal")
	log.Printf("   POST /test/completed")
	log.Printf("   POST /test/hangup")
	log.Printf("   POST /test/optout")
	log.Printf("   POST /test/appointment")
	log.Printf("")
	
	// Log configuration status
	if config.HasPipedriveConfig() {
		log.Printf("✅ Pipedrive API configured (real integration mode)")
	} else {
		log.Printf("⚠️  Pipedrive API not configured (simulation mode)")
		log.Printf("   Set PIPEDRIVE_API_KEY to enable real Pipedrive integration")
	}
	
	log.Printf("🔧 Visit http://localhost:8080 to test the webhooks!")
	log.Printf("📖 See README.md for detailed usage instructions")

	// Test endpoint for call_analyzed webhook
	router.POST("/test/call-analyzed", func(c *gin.Context) {
		testData := RetellCallAnalyzedPayload{
			Event: "call_analyzed",
			Call: struct {
				CallID                    string `json:"call_id"`
				CallType                  string `json:"call_type"`
				AgentID                   string `json:"agent_id"`
				AgentVersion              int    `json:"agent_version"`
				AgentName                 string `json:"agent_name"`
				CollectedDynamicVariables struct {
					CurrentAgentState string `json:"current_agent_state"`
				} `json:"collected_dynamic_variables"`
				CallStatus                string `json:"call_status"`
				StartTimestamp            int64  `json:"start_timestamp"`
				EndTimestamp              int64  `json:"end_timestamp"`
				DurationMs                int    `json:"duration_ms"`
				Transcript                string `json:"transcript"`
				DisconnectionReason       string `json:"disconnection_reason"`
				CallAnalysis              struct {
					CallSummary         string `json:"call_summary"`
					InVoicemail         bool   `json:"in_voicemail"`
					UserSentiment       string `json:"user_sentiment"`
					CallSuccessful      bool   `json:"call_successful"`
					CustomAnalysisData  map[string]interface{} `json:"custom_analysis_data"`
				} `json:"call_analysis"`
				RecordingURL              string `json:"recording_url"`
				RecordingMultiChannelURL  string `json:"recording_multi_channel_url"`
				PublicLogURL              string `json:"public_log_url"`
			}{
				CallID:       "test-analyzed-" + strconv.FormatInt(time.Now().Unix(), 10),
				CallType:     "web_call",
				AgentID:      "agent_test123",
				AgentVersion: 1,
				AgentName:    "Test Agent",
				CollectedDynamicVariables: struct {
					CurrentAgentState string `json:"current_agent_state"`
				}{
					CurrentAgentState: "greeting",
				},
				CallStatus:          "ended",
				StartTimestamp:      time.Now().Add(-5 * time.Minute).UnixMilli(),
				EndTimestamp:        time.Now().UnixMilli(),
				DurationMs:          300000, // 5 minutes
				Transcript:          "User: Hello?\nAgent: Hi there! This is a test call from our AI agent. How can I help you today?\nUser: I'm interested in your services.\nAgent: Great! Let me tell you about our amazing services...",
				DisconnectionReason: "user_hangup",
				CallAnalysis: struct {
					CallSummary         string `json:"call_summary"`
					InVoicemail         bool   `json:"in_voicemail"`
					UserSentiment       string `json:"user_sentiment"`
					CallSuccessful      bool   `json:"call_successful"`
					CustomAnalysisData  map[string]interface{} `json:"custom_analysis_data"`
				}{
					CallSummary:    "The user showed interest in our services during this test call. The conversation was brief but positive.",
					InVoicemail:    false,
					UserSentiment:  "Positive",
					CallSuccessful: true,
					CustomAnalysisData: map[string]interface{}{
						"interest_level": "high",
						"follow_up_needed": true,
					},
				},
				RecordingURL:             "https://example.com/recording.wav",
				RecordingMultiChannelURL: "https://example.com/recording_multichannel.wav",
				PublicLogURL:             "https://example.com/public.log",
			},
		}

		if err := pipedriveService.ProcessRetellCallAnalyzed(testData); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Test failed: " + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Test call_analyzed sent successfully!",
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
				PersonID:   139, // Use existing person ID
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

	// Start server
	address := config.Host + ":" + config.Port
	if err := router.Run(address); err != nil {
		log.Fatal("Failed to start server:", err)
	}
	}
}
