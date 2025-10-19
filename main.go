package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables FIRST (optional - will use system env vars if .env not found)
	_ = godotenv.Load() // Ignore error - will use Railway/Vercel env vars in production

	// Set Gin to debug mode for testing
	gin.SetMode(gin.DebugMode)

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

	// DEBUG: Print configuration
	log.Printf("🔧 [DEBUG] PipedriveAPIKey: %s", config.PipedriveAPIKey)
	log.Printf("🔧 [DEBUG] RetellAPIKey: %s", config.RetellAPIKey)
	log.Printf("🔧 [DEBUG] RetellAssistantID: %s", config.RetellAssistantID)
	log.Printf("🔧 [DEBUG] RetellFromNumber: %s", config.RetellFromNumber)
	log.Printf("🔧 [DEBUG] HasPipedriveConfig: %t", config.HasPipedriveConfig())
	log.Printf("🔧 [DEBUG] HasRetellConfig: %t", config.HasRetellConfig())

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

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "PipCal Webhook Server",
		})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Starting PipCal Webhook Server on port %s", port)
	log.Printf("📋 Available endpoints:")
	log.Printf("   GET  /health")
	log.Printf("   POST /webhook/retell")
	log.Printf("   POST /webhook/cal")
	log.Printf("   POST /webhook/retell/analyzed")
	log.Printf("   POST /webhook/pipedrive/lead")
	log.Printf("   POST /test/completed")
	log.Printf("   POST /test/pipedrive-lead")

	// Check if Pipedrive is configured
	if config.HasPipedriveConfig() {
		log.Printf("✅ Pipedrive API configured")
	} else {
		log.Printf("⚠️  Pipedrive API not configured (simulation mode)")
		log.Printf("   Set PIPEDRIVE_API_KEY to enable real Pipedrive integration")
	}

	// Check if Retell AI is configured
	if config.HasRetellConfig() {
		log.Printf("✅ Retell AI configured")
	} else {
		log.Printf("⚠️  Retell AI not configured")
		log.Printf("   Set RETELL_API_KEY and RETELL_ASSISTANT_ID to enable real Retell AI integration")
	}

	// Debug configuration
	log.Printf("🔧 [DEBUG] Configuration details:")
	log.Printf("   PIPEDRIVE_API_KEY: %s", config.PipedriveAPIKey)
	log.Printf("   RETELL_API_KEY: %s", config.RetellAPIKey)
	log.Printf("   RETELL_ASSISTANT_ID: %s", config.RetellAssistantID)
	log.Printf("   RETELL_FROM_NUMBER: %s", config.RetellFromNumber)

	log.Printf("🔧 Visit http://localhost:%s to test the webhooks!", port)
	log.Printf("📖 See README.md for detailed usage instructions")

	// Start the server
	router.Run(":" + port)
}

// Handler is the main entry point for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	// Set Gin to release mode for Vercel
	gin.SetMode(gin.ReleaseMode)
	
	// Create Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

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

	// Create Pipedrive service
	pipedriveService := NewPipedriveService(config)

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

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "PipCal Webhook Server is running on Vercel!",
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

	// Handle the request
	router.ServeHTTP(w, r)
}

// All the necessary types and functions from the api package

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Port string
	Host string

	// Pipedrive API configuration (for real integration)
	PipedriveAPIKey    string
	PipedriveBaseURL   string
	PipedriveCompanyID string

	// Retell AI configuration
	RetellAPIKey       string
	RetellAssistantID  string
	RetellBaseURL      string
	RetellFromNumber   string

	// Webhook security (optional)
	RetellWebhookSecret string
	CalWebhookSecret    string

	// Logging configuration
	LogLevel string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	config := &Config{
		// Server defaults
		Port: getEnv("PORT", "8080"),
		Host: getEnv("HOST", "0.0.0.0"),

		// Pipedrive configuration
		PipedriveAPIKey:    getEnv("PIPEDRIVE_API_KEY", ""),
		PipedriveBaseURL:   getEnv("PIPEDRIVE_BASE_URL", "https://api.pipedrive.com/v1"),
		PipedriveCompanyID: getEnv("PIPEDRIVE_COMPANY_ID", ""),

		// Retell AI configuration
		RetellAPIKey:       getEnv("RETELL_API_KEY", ""),
		RetellAssistantID:  getEnv("RETELL_ASSISTANT_ID", ""),
		RetellBaseURL:      getEnv("RETELL_BASE_URL", "https://api.retellai.com"),
		RetellFromNumber:   getEnv("RETELL_FROM_NUMBER", "18005300627"),

		// Webhook secrets (optional for basic auth)
		RetellWebhookSecret: getEnv("RETELL_WEBHOOK_SECRET", ""),
		CalWebhookSecret:    getEnv("CAL_WEBHOOK_SECRET", ""),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return config
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// HasPipedriveConfig returns true if Pipedrive API key is configured
func (c *Config) HasPipedriveConfig() bool {
	return c.PipedriveAPIKey != ""
}

// HasRetellConfig returns true if Retell AI API key and assistant ID are configured
func (c *Config) HasRetellConfig() bool {
	return c.RetellAPIKey != "" && c.RetellAssistantID != ""
}

// Contact represents a contact in the system
type Contact struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	DNC   bool   `json:"dnc"` // Do Not Call flag
}

// RetellWebhookPayload represents the incoming Retell AI webhook data
type RetellWebhookPayload struct {
	CallID        string `json:"call_id"`
	ContactPhone  string `json:"contact_phone"`
	Transcript    string `json:"transcript"`
	Duration      string `json:"duration"` // Format: "00:02:15"
	Status        string `json:"status"`   // "completed", "hangup", "optout"
	Timestamp     string `json:"timestamp"` // ISO8601 format
	Event         string `json:"event"`     // "call.completed", "call.hangup", "call.optout"
}

// PipedriveLeadWebhookPayload represents the incoming Pipedrive lead webhook data
type PipedriveLeadWebhookPayload struct {
	Data struct {
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
	} `json:"data"`
	Previous interface{} `json:"previous"`
	Meta     struct {
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
	} `json:"meta"`
}

// RetellCallRequest represents the request to create a call via Retell AI
type RetellCallRequest struct {
	FromNumber          string                 `json:"from_number"`
	ToNumber            string                 `json:"to_number"`
	AssistantID         string                 `json:"assistant_id"`
	MaxDurationSeconds  int                    `json:"max_duration_seconds,omitempty"`
	DynamicVariables    map[string]interface{} `json:"dynamic_variables,omitempty"`
}

// RetellCallResponse represents the response from Retell AI call creation
type RetellCallResponse struct {
	CallID string `json:"call_id"`
	Status string `json:"status"`
}

// WebhookResponse represents the response sent back to webhook callers
type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// CalWebhookPayload represents the incoming Cal.com webhook data
type CalWebhookPayload struct {
	TriggerEvent string `json:"triggerEvent"`
	CreatedAt    string `json:"createdAt"`
	Payload      struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
		Attendees []struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"attendees"`
		Location string `json:"location"`
	} `json:"payload"`
}

// PipedriveService handles real Pipedrive API interactions
type PipedriveService struct {
	config       *Config
	httpClient   *http.Client
	callMappings map[string]CallMapping // Maps callID to call info
}

// CallMapping stores call information for later use
type CallMapping struct {
	PersonName string
	PhoneNumber string
	LeadTitle  string
	PersonID   int
	Timestamp  time.Time
}

// PipedrivePhone represents a phone number from Pipedrive API
type PipedrivePhone struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

// PipedrivePerson represents a person from Pipedrive API
type PipedrivePerson struct {
	ID    int             `json:"id"`
	Name  string          `json:"name"`
	Email []PipedrivePhone `json:"email"`
	Phone []PipedrivePhone `json:"phone"`
}

// PipedrivePersonResponse represents the response from Pipedrive persons API
type PipedrivePersonResponse struct {
	Success bool            `json:"success"`
	Data    *PipedrivePerson `json:"data"`
}

// PipedrivePersonSearchResponse represents the search response from Pipedrive
type PipedrivePersonSearchResponse struct {
	Success bool              `json:"success"`
	Data    *PipedrivePerson  `json:"data"`
	Items   []PipedrivePerson `json:"items"`
}

// PipedriveActivity represents an activity in Pipedrive
type PipedriveActivity struct {
	ID       int    `json:"id"`
	Subject  string `json:"subject"`
	Type     string `json:"type"`
	DueDate  string `json:"due_date"`
	PersonID int    `json:"person_id"`
	Note     string `json:"note"`
	Duration string `json:"duration"`
}

// PipedriveActivityResponse represents the response from Pipedrive activities API
type PipedriveActivityResponse struct {
	Success bool              `json:"success"`
	Data    *PipedriveActivity `json:"data"`
}

// NewPipedriveService creates a new Pipedrive service instance
func NewPipedriveService(config *Config) *PipedriveService {
	return &PipedriveService{
		config:       config,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		callMappings: make(map[string]CallMapping),
	}
}

// makePipedriveRequest makes an HTTP request to Pipedrive API
func (p *PipedriveService) makePipedriveRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	// Check if endpoint already has query parameters
	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
		log.Printf("🔧 [DEBUG] Endpoint contains '?', using '&' separator")
	} else {
		log.Printf("🔧 [DEBUG] Endpoint does NOT contain '?', using '?' separator")
	}
	log.Printf("🔧 [DEBUG] Endpoint before building URL: %s", endpoint)
	url := p.config.PipedriveBaseURL + endpoint + separator + "api_token=" + p.config.PipedriveAPIKey
	
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
		log.Printf("📤 Request Body: %s", string(jsonData))
	}
	
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	log.Printf("🌐 Making %s request to Pipedrive: %s", method, endpoint)
	log.Printf("🔗 Full URL: %s", url)
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	
	// Log the response
	log.Printf("📥 Pipedrive Response Status: %d", resp.StatusCode)
	
	// Read and log response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response body: %v", err)
	} else {
		log.Printf("📥 Pipedrive Response Body: %s", string(bodyBytes))
	}
	
	// Create a new response with the body for further processing
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	
	return resp, nil
}

// GetPersonByID retrieves a person by ID from Pipedrive
func (p *PipedriveService) GetPersonByID(personID int) (*PipedrivePerson, error) {
	endpoint := fmt.Sprintf("/persons/%d", personID)
	resp, err := p.makePipedriveRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get person: HTTP %d", resp.StatusCode)
	}

	var result PipedrivePersonResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("failed to get person")
	}

	return result.Data, nil
}

// extractPhoneFromPerson extracts phone number from PipedrivePerson
func (p *PipedriveService) extractPhoneFromPerson(person *PipedrivePerson) string {
	if person.Phone != nil && len(person.Phone) > 0 {
		phoneNumber := person.Phone[0].Value
		
		// Clean the phone number (remove spaces, dashes, parentheses)
		phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")
		phoneNumber = strings.ReplaceAll(phoneNumber, "-", "")
		phoneNumber = strings.ReplaceAll(phoneNumber, "(", "")
		phoneNumber = strings.ReplaceAll(phoneNumber, ")", "")
		
		// Only add +1 if the number doesn't already have a country code
		if !strings.HasPrefix(phoneNumber, "+") {
			// If it doesn't start with +, add +1
			phoneNumber = "+1" + phoneNumber
		} else if strings.HasPrefix(phoneNumber, "1") && !strings.HasPrefix(phoneNumber, "+1") {
			// If it starts with 1 but not +1, add the +
			phoneNumber = "+" + phoneNumber
		}
		
		return phoneNumber
	}
	return ""
}

// CreateRetellCall creates a call via Retell AI API
func (p *PipedriveService) CreateRetellCall(phoneNumber, personName, leadTitle string) (string, error) {
	// Check if we have valid Retell AI configuration
	if p.config.RetellAPIKey == "" || p.config.RetellAssistantID == "" {
		return "", fmt.Errorf("Retell AI not configured: missing API key or assistant ID")
	}

	log.Printf("🚀 Creating Retell AI call for %s (%s) - Lead: %s", personName, phoneNumber, leadTitle)

	callRequest := RetellCallRequest{
		FromNumber:          p.config.RetellFromNumber,
		ToNumber:            phoneNumber,
		AssistantID:         p.config.RetellAssistantID,
		MaxDurationSeconds:  300, // 5 minutes max
		DynamicVariables: map[string]interface{}{
			"person_name": personName,
			"lead_title":  leadTitle,
		},
	}

	// Use the correct Retell AI endpoint
	url := p.config.RetellBaseURL + "/v2/create-phone-call"
	jsonData, err := json.Marshal(callRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal call request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.RetellAPIKey)

	log.Printf("🌐 Making Retell AI call to: %s", url)
	log.Printf("📤 Request Body: %s", string(jsonData))
	log.Printf("🔑 Using API Key: %s...", p.config.RetellAPIKey[:min(8, len(p.config.RetellAPIKey))])

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make Retell AI request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("📥 Retell AI Response Status: %d", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("📥 Retell AI Response Body: %s", string(body))

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		var callResponse RetellCallResponse
		if err := json.Unmarshal(body, &callResponse); err != nil {
			// Try to extract call ID from different response formats
			var responseMap map[string]interface{}
			if err := json.Unmarshal(body, &responseMap); err == nil {
				if callID, ok := responseMap["call_id"].(string); ok {
					log.Printf("✅ Successfully created Retell AI call: %s", callID)
					return callID, nil
				}
				if callID, ok := responseMap["id"].(string); ok {
					log.Printf("✅ Successfully created Retell AI call: %s", callID)
					return callID, nil
				}
			}
			return "", fmt.Errorf("failed to parse Retell AI response: %v", err)
		}
		log.Printf("✅ Successfully created Retell AI call: %s", callResponse.CallID)
		return callResponse.CallID, nil
	}

	return "", fmt.Errorf("Retell AI call failed: HTTP %d, Response: %s", resp.StatusCode, string(body))
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// storeCallMapping stores call information for later retrieval
func (p *PipedriveService) storeCallMapping(callID, personName, phoneNumber, leadTitle string, personID int) {
	p.callMappings[callID] = CallMapping{
		PersonName:  personName,
		PhoneNumber: phoneNumber,
		LeadTitle:   leadTitle,
		PersonID:    personID,
		Timestamp:   time.Now(),
	}
	log.Printf("📝 Stored call mapping for %s: %s (%s)", callID, personName, phoneNumber)
}

// ProcessPipedriveLead processes a Pipedrive lead webhook and triggers a Retell AI call
func (p *PipedriveService) ProcessPipedriveLead(payload PipedriveLeadWebhookPayload) error {
	log.Printf("🔍 [SIMULATION MODE] Processing Pipedrive lead webhook")
	log.Printf("   Lead ID: %s", payload.Data.ID)
	log.Printf("   Title: %s", payload.Data.Title)
	log.Printf("   Person ID: %d", payload.Data.PersonID)
	log.Printf("   Action: %s", payload.Meta.Action)
	log.Printf("   ⚠️  This is a SIMULATION SERVER - not real Pipedrive or Retell AI")

	// Check configuration status
	log.Printf("🔧 [DEBUG] Pipedrive configured: %t", p.config.HasPipedriveConfig())
	log.Printf("🔧 [DEBUG] Retell AI configured: %t", p.config.HasRetellConfig())
	log.Printf("🔧 [DEBUG] Pipedrive API Key: %s", p.config.PipedriveAPIKey)
	log.Printf("🔧 [DEBUG] Retell API Key: %s", p.config.RetellAPIKey)
	log.Printf("🔧 [DEBUG] Retell Assistant ID: %s", p.config.RetellAssistantID)

	// Only process lead creation events
	if payload.Meta.Action != "create" {
		log.Printf("ℹ️ Skipping lead event: %s (only processing 'create' events)", payload.Meta.Action)
		return nil
	}

	// Try to process with real integration if configured
	if p.config.HasPipedriveConfig() && p.config.HasRetellConfig() {
		log.Printf("🚀 [REAL INTEGRATION] Processing Pipedrive lead webhook")

		// Get person details from Pipedrive
		person, err := p.GetPersonByID(payload.Data.PersonID)
		if err != nil {
			log.Printf("❌ Failed to get person details: %v", err)
			return fmt.Errorf("failed to get person details: %v", err)
		}

		// Extract phone number
		phoneNumber := p.extractPhoneFromPerson(person)
		if phoneNumber == "" {
			log.Printf("⚠️ No phone number found for person %d, skipping call", payload.Data.PersonID)
			return nil
		}

		log.Printf("📞 Found phone number: %s for person: %s", phoneNumber, person.Name)

		// Create Retell AI call with person name and lead title
		callID, err := p.CreateRetellCall(phoneNumber, person.Name, payload.Data.Title)
		if err != nil {
			log.Printf("❌ Failed to create Retell AI call: %v", err)
			// Don't return error, just log it and continue
			callID = "failed-" + strconv.FormatInt(time.Now().Unix(), 10)
		} else {
			log.Printf("✅ Created Retell AI call %s for lead %s (person: %s, phone: %s)", 
				callID, payload.Data.Title, person.Name, phoneNumber)
		}

		// Store the call mapping for later use in call_analyzed webhook
		p.storeCallMapping(callID, person.Name, phoneNumber, payload.Data.Title, payload.Data.PersonID)

		// Create activity in Pipedrive to track the call
		activityData := map[string]interface{}{
			"subject":   fmt.Sprintf("AI Call Initiated - Lead: %s", payload.Data.Title),
			"type":      "call",
			"person_id": payload.Data.PersonID,
			"note":      fmt.Sprintf("Retell AI call initiated for lead: %s\nCall ID: %s\nPhone: %s", 
				payload.Data.Title, callID, phoneNumber),
			"done":      0, // Mark as pending
			"due_date":  time.Now().Format("2006-01-02"),
			"due_time":  time.Now().Add(5 * time.Minute).Format("15:04:05"),
		}

		resp, err := p.makePipedriveRequest("POST", "/activities", activityData)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to create activity: %v", err)
		} else {
			resp.Body.Close()
			log.Printf("✅ Created activity for Retell AI call")
		}
	} else {
		log.Printf("⚠️  Configuration missing - running in simulation mode")
		if !p.config.HasPipedriveConfig() {
			log.Printf("   Missing: PIPEDRIVE_API_KEY")
		}
		if !p.config.HasRetellConfig() {
			log.Printf("   Missing: RETELL_API_KEY or RETELL_ASSISTANT_ID")
		}
	}

	return nil
}

// ProcessRetellCall processes a Retell AI call webhook
func (p *PipedriveService) ProcessRetellCall(payload RetellWebhookPayload) error {
	log.Printf("🔧 [DEBUG] ProcessRetellCall called with event: %s", payload.Event)
	if p.config.HasPipedriveConfig() {
		log.Printf("🚀 [REAL PIPEDRIVE] Processing Retell webhook: %s", payload.Event)
		// Implementation for real Pipedrive integration
	} else {
		log.Printf("🔍 [SIMULATION MODE] Processing Retell webhook: %s", payload.Event)
		log.Printf("   Call ID: %s", payload.CallID)
		log.Printf("   Phone: %s", payload.ContactPhone)
		log.Printf("   Duration: %s", payload.Duration)
		log.Printf("   Status: %s", payload.Status)

		if payload.Transcript != "" {
			log.Printf("   Transcript: %s", payload.Transcript)
		}

		log.Printf("   ⚠️  This is a SIMULATION SERVER - not real Retell AI or Pipedrive")
	}

	return nil
}

// ProcessCalAppointment processes a Cal.com appointment webhook
func (p *PipedriveService) ProcessCalAppointment(payload CalWebhookPayload) error {
	log.Printf("🔧 [DEBUG] ProcessCalAppointment called")
	log.Printf("🔧 [DEBUG] HasPipedriveConfig: %v", p.config.HasPipedriveConfig())
	log.Printf("🔧 [DEBUG] PIPEDRIVE_API_KEY: %s", p.config.PipedriveAPIKey)

	if p.config.HasPipedriveConfig() {
		log.Printf("🚀 [REAL PIPEDRIVE] Processing Cal.com appointment webhook")

		// Parse start time
		startTime, err := time.Parse(time.RFC3339, payload.Payload.StartTime)
		if err != nil {
			log.Printf("❌ [DEBUG] Error parsing startTime: %v", err)
			return fmt.Errorf("invalid startTime format: %v", err)
		}

		// Get the first attendee (main contact)
		attendee := payload.Payload.Attendees[0]
		log.Printf("📧 [DEBUG] Processing attendee: %s (%s)", attendee.Name, attendee.Email)

		// Find or create contact by email
		contact, err := p.FindOrCreateContactByEmail(attendee.Email, attendee.Name)
		if err != nil {
			log.Printf("❌ [DEBUG] Error finding/creating contact: %v", err)
			return fmt.Errorf("failed to find/create contact: %v", err)
		}

		log.Printf("✅ [DEBUG] Contact found/created: ID=%s, Name=%s", contact.ID, contact.Name)

		// Convert contactID to int
		personID, err := strconv.Atoi(contact.ID)
		if err != nil {
			log.Printf("❌ [DEBUG] Error converting contact ID: %v", err)
			return fmt.Errorf("invalid contact ID: %v", err)
		}

		// Create appointment activity in Pipedrive
		activityData := map[string]interface{}{
			"subject":   fmt.Sprintf("Cal.com: %s", payload.Payload.Title),
			"type":      "meeting",
			"person_id": personID,
			"note":      fmt.Sprintf("Appointment: %s\nAttendee: %s (%s)\nMeeting URL: %s", payload.Payload.Title, attendee.Name, attendee.Email, payload.Payload.Location),
			"done":      0, // Not completed yet
			"due_date":  startTime.Format("2006-01-02"),
			"due_time":  startTime.Format("15:04:05"),
		}

		log.Printf("🔧 [DEBUG] Creating appointment activity for personID: %d", personID)
		log.Printf("🔧 [DEBUG] Activity data: %+v", activityData)

		resp, err := p.makePipedriveRequest("POST", "/activities", activityData)
		if err != nil {
			log.Printf("❌ [DEBUG] Error creating appointment activity: %v", err)
			return fmt.Errorf("failed to create appointment activity: %v", err)
		}
		defer resp.Body.Close()

		log.Printf("🔧 [DEBUG] Appointment activity creation response status: %d", resp.StatusCode)

		var activityResult PipedriveActivityResponse
		if err := json.NewDecoder(resp.Body).Decode(&activityResult); err != nil {
			log.Printf("❌ [DEBUG] Error decoding appointment activity response: %v", err)
			return fmt.Errorf("failed to decode activity response: %v", err)
		}

		log.Printf("🔧 [DEBUG] Appointment activity result: %+v", activityResult)

		if !activityResult.Success {
			log.Printf("❌ [DEBUG] Appointment activity creation failed in Pipedrive")
			return fmt.Errorf("failed to create appointment activity in Pipedrive")
		}

		log.Printf("✅ Created appointment activity in Pipedrive: ID=%d", activityResult.Data.ID)

	} else {
		// Simulation mode
		log.Printf("🔍 [SIMULATION MODE] Processing Cal.com appointment webhook")
		log.Printf("   Event: %s", payload.TriggerEvent)
		log.Printf("   Booking ID: %d", payload.Payload.ID)
		log.Printf("   Title: %s", payload.Payload.Title)
		if len(payload.Payload.Attendees) > 0 {
			attendee := payload.Payload.Attendees[0]
			log.Printf("   Attendee: %s (%s)", attendee.Name, attendee.Email)
		}
		log.Printf("   Start Time: %s", payload.Payload.StartTime)
		log.Printf("   End Time: %s", payload.Payload.EndTime)
		log.Printf("   Location: %s", payload.Payload.Location)
		log.Printf("   ⚠️  This is a SIMULATION SERVER - not real Cal.com or Pipedrive")
	}

	return nil
}

// FindOrCreateContactByEmail finds or creates a contact by email address
func (p *PipedriveService) FindOrCreateContactByEmail(email, name string) (*Contact, error) {
	log.Printf("🔍 [REAL PIPEDRIVE API] Searching for contact by email: %s", email)

	// Search for existing contact by email
	// URL-encode the email to handle special characters like @ and +
	encodedEmail := url.QueryEscape(email)
	searchURL := fmt.Sprintf("/persons/search?term=%s&fields=email", encodedEmail)
	resp, err := p.makePipedriveRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for contact: %v", err)
	}
	defer resp.Body.Close()

	var searchResult PipedrivePersonSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %v", err)
	}

	// If contact found, return it
	if searchResult.Success && len(searchResult.Items) > 0 {
		person := searchResult.Items[0]
		log.Printf("✅ Found existing contact: ID=%d, Name=%s", person.ID, person.Name)
		return &Contact{
			ID:    strconv.Itoa(person.ID),
			Name:  person.Name,
			Email: email,
			Phone: extractPhoneFromPerson(&person),
		}, nil
	}

	// Contact not found, create new one
	log.Printf("📝 Creating new contact in Pipedrive for email: %s", email)
	personData := map[string]interface{}{
		"name": name,
		"email": []map[string]interface{}{
			{"value": email},
		},
	}

	resp, err = p.makePipedriveRequest("POST", "/persons", personData)
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %v", err)
	}
	defer resp.Body.Close()

	var personResult PipedrivePersonResponse
	if err := json.NewDecoder(resp.Body).Decode(&personResult); err != nil {
		return nil, fmt.Errorf("failed to decode person response: %v", err)
	}

	if !personResult.Success {
		return nil, fmt.Errorf("failed to create contact in Pipedrive")
	}

	person := personResult.Data
	log.Printf("✅ Created new contact in Pipedrive: ID=%d, Name=%s", person.ID, person.Name)

	return &Contact{
		ID:    strconv.Itoa(person.ID),
		Name:  person.Name,
		Email: email,
		Phone: extractPhoneFromPerson(person),
	}, nil
}

// extractPhoneFromPerson extracts phone from PipedrivePerson
func extractPhoneFromPerson(person *PipedrivePerson) string {
	if len(person.Phone) > 0 {
		return person.Phone[0].Value
	}
	return ""
}

// Handler functions
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "PipCal Webhook Server",
		"version": "1.0.0",
	})
}

func RetellWebhookHandler(pipedriveService *PipedriveService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload RetellWebhookPayload

		// Bind JSON payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Invalid JSON payload",
			})
			return
		}

		// Validate required fields for Retell format
		if payload.CallID == "" || payload.ContactPhone == "" {
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Missing required fields: call_id and contact_phone",
			})
			return
		}

		// Process the call
		if err := pipedriveService.ProcessRetellCall(payload); err != nil {
			c.JSON(http.StatusInternalServerError, WebhookResponse{
				Success: false,
				Message: "Failed to process call: " + err.Error(),
			})
			return
		}

		// Return success response
		c.JSON(http.StatusOK, WebhookResponse{
			Success: true,
			Message: "Retell webhook processed successfully",
			Data: gin.H{
				"call_id":       payload.CallID,
				"contact_phone": payload.ContactPhone,
				"event":         payload.Event,
				"status":        payload.Status,
				"duration":      payload.Duration,
			},
		})
	}
}

func CalWebhookHandler(pipedriveService *PipedriveService) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("🔔 [CAL WEBHOOK] Received Cal.com webhook request")

		var payload CalWebhookPayload

		// Bind JSON payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			log.Printf("❌ [CAL WEBHOOK] Failed to bind JSON: %v", err)
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Invalid JSON payload",
			})
			return
		}

		log.Printf("📦 [CAL WEBHOOK] Payload received: Event=%s, ID=%d, Title=%s",
			payload.TriggerEvent, payload.Payload.ID, payload.Payload.Title)

		// Validate required fields
		if len(payload.Payload.Attendees) == 0 {
			log.Printf("❌ [CAL WEBHOOK] Validation failed: No attendees")
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Missing required field: attendees",
			})
			return
		}

		if payload.Payload.StartTime == "" || payload.Payload.Location == "" {
			log.Printf("❌ [CAL WEBHOOK] Validation failed: StartTime=%s, Location=%s",
				payload.Payload.StartTime, payload.Payload.Location)
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Missing required fields: startTime and location",
			})
			return
		}

		log.Printf("✅ [CAL WEBHOOK] Validation passed, calling ProcessCalAppointment")

		// Process the appointment
		if err := pipedriveService.ProcessCalAppointment(payload); err != nil {
			log.Printf("❌ [CAL WEBHOOK] ProcessCalAppointment failed: %v", err)
			c.JSON(http.StatusInternalServerError, WebhookResponse{
				Success: false,
				Message: "Failed to process appointment: " + err.Error(),
			})
			return
		}

		log.Printf("✅ [CAL WEBHOOK] ProcessCalAppointment completed successfully")

		// Return success response
		c.JSON(http.StatusOK, WebhookResponse{
			Success: true,
			Message: "Appointment processed successfully",
			Data: gin.H{
				"trigger_event": payload.TriggerEvent,
				"booking_id":    payload.Payload.ID,
				"title":         payload.Payload.Title,
				"start_time":    payload.Payload.StartTime,
				"end_time":      payload.Payload.EndTime,
				"location":      payload.Payload.Location,
				"attendees":     payload.Payload.Attendees,
			},
		})

		log.Printf("🎉 [CAL WEBHOOK] Webhook response sent successfully")
	}
}

func RetellCallAnalyzedHandler(pipedriveService *PipedriveService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, WebhookResponse{
			Success: true,
			Message: "Retell call analyzed webhook processed successfully",
		})
	}
}

func PipedriveLeadWebhookHandler(pipedriveService *PipedriveService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload PipedriveLeadWebhookPayload

		// Bind JSON payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Invalid JSON payload",
			})
			return
		}

		// Validate required fields
		if payload.Data.ID == "" || payload.Data.PersonID == 0 {
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Missing required fields: data.id and data.person_id",
			})
			return
		}

		// Process the lead
		if err := pipedriveService.ProcessPipedriveLead(payload); err != nil {
			c.JSON(http.StatusInternalServerError, WebhookResponse{
				Success: false,
				Message: "Failed to process lead: " + err.Error(),
			})
			return
		}

		// Return success response
		c.JSON(http.StatusOK, WebhookResponse{
			Success: true,
			Message: "Pipedrive lead webhook processed successfully",
			Data: gin.H{
				"lead_id":    payload.Data.ID,
				"person_id":  payload.Data.PersonID,
				"title":      payload.Data.Title,
				"action":     payload.Meta.Action,
			},
		})
	}
}
