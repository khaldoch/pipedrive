package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CallMapping stores call information for later use
type CallMapping struct {
	PersonName string
	PhoneNumber string
	LeadTitle  string
	PersonID   int
	Timestamp  time.Time
}

// PipedriveService handles real Pipedrive API interactions
type PipedriveService struct {
	config       *Config
	httpClient   *http.Client
	callMappings map[string]CallMapping // Maps callID to call info
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

// PipedriveLead represents a lead from Pipedrive API
type PipedriveLead struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	PersonID  int    `json:"person_id"`
	OwnerID   int    `json:"owner_id"`
	AddTime   string `json:"add_time"`
	UpdateTime string `json:"update_time"`
}

// PipedriveLeadSearchResponse represents the search response for leads from Pipedrive
type PipedriveLeadSearchResponse struct {
	Success bool            `json:"success"`
	Data    *PipedriveLead  `json:"data"`
	Items   []PipedriveLead `json:"items"`
}

// PipedriveActivity represents an activity in Pipedrive
type PipedriveActivity struct {
	ID          int    `json:"id"`
	Subject     string `json:"subject"`
	Type        string `json:"type"`
	DueDate     string `json:"due_date"`
	PersonID    int    `json:"person_id"`
	Note        string `json:"note"`
	Duration    string `json:"duration"`
	MeetingURL  string `json:"meeting_url,omitempty"`
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
	}
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

// FindOrCreateContact finds or creates a contact in Pipedrive
func (p *PipedriveService) FindOrCreateContact(contactData ContactPayload) (*Contact, error) {
	if p.config.HasPipedriveConfig() {
		// REAL PIPEDRIVE INTEGRATION
		log.Printf("🔍 [REAL PIPEDRIVE API] Searching for contact: %s (%s)", contactData.Name, contactData.Email)
		
		// 1. Search for existing contact by email
		searchEndpoint := fmt.Sprintf("/persons/search?term=%s&fields=email", contactData.Email)
		resp, err := p.makePipedriveRequest("GET", searchEndpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to search contact: %v", err)
		}
		defer resp.Body.Close()
		
		var searchResult PipedrivePersonSearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
			return nil, fmt.Errorf("failed to decode search response: %v", err)
		}
		
		// If contact found, return it
		if searchResult.Success && len(searchResult.Items) > 0 {
			person := searchResult.Items[0]
			phone := ""
			email := ""
			if len(person.Phone) > 0 {
				phone = person.Phone[0].Value
			}
			if len(person.Email) > 0 {
				email = person.Email[0].Value
			}
			log.Printf("✅ Found existing contact in Pipedrive: ID=%d, Name=%s", person.ID, person.Name)
			return &Contact{
				ID:    fmt.Sprintf("%d", person.ID),
				Name:  person.Name,
				Email: email,
				Phone: phone,
				DNC:   false,
			}, nil
		}
		
		// 2. Create new contact if not found
		log.Printf("📝 Creating new contact in Pipedrive: %s", contactData.Name)
		createData := map[string]interface{}{
			"name":  contactData.Name,
			"email": contactData.Email,
			"phone": contactData.Phone,
		}
		
		resp, err = p.makePipedriveRequest("POST", "/persons", createData)
		if err != nil {
			return nil, fmt.Errorf("failed to create contact: %v", err)
		}
		defer resp.Body.Close()
		
		var createResult PipedrivePersonResponse
		if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
			return nil, fmt.Errorf("failed to decode create response: %v", err)
		}
		
		if !createResult.Success || createResult.Data == nil {
			return nil, fmt.Errorf("failed to create contact in Pipedrive")
		}
		
		person := createResult.Data
		phone := ""
		email := ""
		if len(person.Phone) > 0 {
			phone = person.Phone[0].Value
		}
		if len(person.Email) > 0 {
			email = person.Email[0].Value
		}
		log.Printf("✅ Created new contact in Pipedrive: ID=%d, Name=%s", person.ID, person.Name)
		return &Contact{
			ID:    fmt.Sprintf("%d", person.ID),
			Name:  person.Name,
			Email: email,
			Phone: phone,
			DNC:   false,
		}, nil
		
	} else {
		log.Printf("🔍 [SIMULATION MODE] Processing webhook request for contact: %s (%s)", contactData.Name, contactData.Email)
		log.Printf("   ⚠️  This is a SIMULATION SERVER - not real Retell AI or Pipedrive")
		log.Printf("   📡 You sent a POST request to /webhook/retell")
		log.Printf("   🎭 Server is simulating what would happen with real Retell AI + Pipedrive")
		
		// Simulate contact lookup/creation
		contact := &Contact{
			ID:    uuid.New().String(),
			Name:  contactData.Name,
			Email: contactData.Email,
			Phone: contactData.Phone,
			DNC:   false,
		}
		
		log.Printf("✅ Contact found/created: ID=%s, Name=%s", contact.ID, contact.Name)
		return contact, nil
	}
}

// LogActivity logs an activity in Pipedrive
func (p *PipedriveService) LogActivity(contactID string, activity Activity) error {
	if p.config.HasPipedriveConfig() {
		// REAL PIPEDRIVE INTEGRATION
		log.Printf("📝 [REAL PIPEDRIVE API] Logging activity for contact %s:", contactID)
		
		// Convert contactID to int for Pipedrive API
		personID, err := strconv.Atoi(contactID)
		if err != nil {
			return fmt.Errorf("invalid contact ID: %v", err)
		}
		
		// Prepare activity data for Pipedrive
		activityData := map[string]interface{}{
			"subject":    activity.Description,
			"type":       activity.Type,
			"due_date":   activity.DateTime.Format("2006-01-02 15:04:05"),
			"person_id":  personID,
			"note":       activity.Transcript,
		}
		
		// Add duration if available
		if activity.Duration > 0 {
			activityData["duration"] = activity.Duration
		}
		
		// Add meeting URL if available
		if activity.MeetingURL != "" {
			activityData["meeting_url"] = activity.MeetingURL
		}
		
		// Create activity in Pipedrive
		resp, err := p.makePipedriveRequest("POST", "/activities", activityData)
		if err != nil {
			return fmt.Errorf("failed to create activity: %v", err)
		}
		defer resp.Body.Close()
		
		var activityResult PipedriveActivityResponse
		if err := json.NewDecoder(resp.Body).Decode(&activityResult); err != nil {
			return fmt.Errorf("failed to decode activity response: %v", err)
		}
		
		if !activityResult.Success || activityResult.Data == nil {
			return fmt.Errorf("failed to create activity in Pipedrive")
		}
		
		log.Printf("✅ Created activity in Pipedrive: ID=%d, Type=%s", activityResult.Data.ID, activity.Type)
		
	} else {
		log.Printf("📝 [SIMULATION MODE] Simulating activity logging for contact %s:", contactID)
		log.Printf("   ⚠️  This is a SIMULATION SERVER - not real Retell AI or Pipedrive")
		log.Printf("   📡 You sent a POST request to /webhook/retell")
		log.Printf("   🎭 Server is simulating what would happen with real Retell AI + Pipedrive")
	}
	
	log.Printf("   Type: %s", activity.Type)
	log.Printf("   Description: %s", activity.Description)
	log.Printf("   DateTime: %s", activity.DateTime.Format(time.RFC3339))
	
	if activity.Duration > 0 {
		log.Printf("   Duration: %d minutes", activity.Duration)
	}
	
	if activity.MeetingURL != "" {
		log.Printf("   Meeting URL: %s", activity.MeetingURL)
	}
	
	if activity.Transcript != "" {
		log.Printf("   Transcript: %s", activity.Transcript)
	}
	
	return nil
}

// MarkContactAsDNC marks a contact as Do Not Call in Pipedrive
func (p *PipedriveService) MarkContactAsDNC(contactID string) error {
	if p.config.HasPipedriveConfig() {
		// REAL PIPEDRIVE INTEGRATION
		log.Printf("🚫 [REAL PIPEDRIVE API] Marking contact %s as Do Not Call (DNC)", contactID)
		
		// Convert contactID to int for Pipedrive API
		personID, err := strconv.Atoi(contactID)
		if err != nil {
			return fmt.Errorf("invalid contact ID: %v", err)
		}
		
		// Update contact with DNC flag
		updateData := map[string]interface{}{
			"custom_fields": map[string]interface{}{
				"do_not_call": true,
			},
		}
		
		endpoint := fmt.Sprintf("/persons/%d", personID)
		resp, err := p.makePipedriveRequest("PUT", endpoint, updateData)
		if err != nil {
			return fmt.Errorf("failed to update contact: %v", err)
		}
		defer resp.Body.Close()
		
		var updateResult PipedrivePersonResponse
		if err := json.NewDecoder(resp.Body).Decode(&updateResult); err != nil {
			return fmt.Errorf("failed to decode update response: %v", err)
		}
		
		if !updateResult.Success {
			return fmt.Errorf("failed to mark contact as DNC in Pipedrive")
		}
		
		log.Printf("✅ Marked contact %d as Do Not Call (DNC) in Pipedrive", personID)
		
	} else {
		log.Printf("🚫 [SIMULATION MODE] Simulating DNC marking for contact %s", contactID)
		log.Printf("   ⚠️  This is a SIMULATION SERVER - not real Retell AI or Pipedrive")
		log.Printf("   📡 You sent a POST request to /webhook/retell")
		log.Printf("   🎭 Server is simulating what would happen with real Retell AI + Pipedrive")
	}
	
	return nil
}

// ProcessRetellCall processes a Retell AI call webhook
func (p *PipedriveService) ProcessRetellCall(payload RetellWebhookPayload) error {
	log.Printf("🔧 [DEBUG] ProcessRetellCall called with event: %s", payload.Event)
	if p.config.HasPipedriveConfig() {
		log.Printf("🚀 [REAL PIPEDRIVE] Processing Retell webhook: %s", payload.Event)
		
		// Parse timestamp
		callTime, err := time.Parse(time.RFC3339, payload.Timestamp)
		if err != nil {
			return fmt.Errorf("invalid timestamp format: %v", err)
		}
		
		// Find or create contact by phone
		contact, err := p.FindOrCreateContactByPhone(payload.ContactPhone)
		if err != nil {
			return fmt.Errorf("failed to find/create contact: %v", err)
		}
		
		// Convert contactID to int
		personID, err := strconv.Atoi(contact.ID)
		if err != nil {
			return fmt.Errorf("invalid contact ID: %v", err)
		}
		
		// Handle different event types
		log.Printf("🔧 [DEBUG] Processing event: %s for personID: %d", payload.Event, personID)
		switch payload.Event {
		case "call_started":
			log.Printf("🔧 [DEBUG] Calling handleCallStarted")
			err := p.handleCallStarted(personID, payload, callTime)
			if err != nil {
				log.Printf("❌ [DEBUG] Error in handleCallStarted: %v", err)
				return err
			}
			log.Printf("🔧 [DEBUG] handleCallStarted completed successfully")
			return nil
		case "call_ended":
			log.Printf("🔧 [DEBUG] Calling handleCallEnded")
			err := p.handleCallEnded(personID, payload, callTime)
			if err != nil {
				log.Printf("❌ [DEBUG] Error in handleCallEnded: %v", err)
				return err
			}
			log.Printf("🔧 [DEBUG] handleCallEnded completed successfully")
			return nil
		case "call.completed":
			log.Printf("🔧 [DEBUG] Calling handleCallCompleted")
			err := p.handleCallCompleted(personID, payload, callTime)
			if err != nil {
				log.Printf("❌ [DEBUG] Error in handleCallCompleted: %v", err)
				return err
			}
			log.Printf("🔧 [DEBUG] handleCallCompleted completed successfully")
			return nil
		case "call.hangup":
			log.Printf("🔧 [DEBUG] Calling handleCallHangup")
			err := p.handleCallHangup(personID, payload, callTime)
			if err != nil {
				log.Printf("❌ [DEBUG] Error in handleCallHangup: %v", err)
				return err
			}
			log.Printf("🔧 [DEBUG] handleCallHangup completed successfully")
			return nil
		case "call.optout":
			log.Printf("🔧 [DEBUG] Calling handleCallOptout")
			err := p.handleCallOptout(personID, payload, callTime)
			if err != nil {
				log.Printf("❌ [DEBUG] Error in handleCallOptout: %v", err)
				return err
			}
			log.Printf("🔧 [DEBUG] handleCallOptout completed successfully")
			return nil
		default:
			log.Printf("⚠️ Unknown event type: %s", payload.Event)
			return nil
		}
		
	} else {
		// Simulation mode
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

// ProcessRetellCallAnalyzed processes a Retell AI call_analyzed webhook
func (p *PipedriveService) ProcessRetellCallAnalyzed(payload RetellCallAnalyzedPayload) error {
	if p.config.HasPipedriveConfig() {
		log.Printf("🚀 [REAL PIPEDRIVE] Processing Retell call_analyzed webhook")

		// Convert timestamps to time.Time
		startTime := time.Unix(payload.Call.StartTimestamp/1000, 0)
		endTime := time.Unix(payload.Call.EndTimestamp/1000, 0)
		
		// Convert duration from milliseconds to HH:MM:SS format
		durationSeconds := payload.Call.DurationMs / 1000
		hours := durationSeconds / 3600
		minutes := (durationSeconds % 3600) / 60
		seconds := durationSeconds % 60
		duration := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

		// Get stored call mapping to find person name and details
		callMapping, exists := p.getCallMapping(payload.Call.CallID)
		if !exists {
			log.Printf("⚠️ Warning: No call mapping found for call ID: %s", payload.Call.CallID)
			// Try to find contact by phone number as fallback
			contact, err := p.FindOrCreateContactByPhone("Unknown")
			if err != nil {
				return fmt.Errorf("failed to find/create contact: %v", err)
			}
			
			// Convert contactID to int
			personID, err := strconv.Atoi(contact.ID)
			if err != nil {
				return fmt.Errorf("invalid contact ID: %v", err)
			}
			
			// Update person with call data in custom fields
			if err := p.UpdatePersonWithCallData(personID, payload.Call.Transcript, duration, startTime.Format("2006-01-02")); err != nil {
				log.Printf("⚠️ Warning: Failed to update person with call data: %v", err)
			}
			
			// Create comprehensive call activity
			activityData := map[string]interface{}{
				"subject":   fmt.Sprintf("AI Call Analyzed - %s", payload.Call.AgentName),
				"type":      "call",
				"person_id": personID,
				"duration":  duration,
				"note":      p.buildCallAnalyzedNote(payload, startTime, endTime, duration),
				"done":      1,
				"due_date":  startTime.Format("2006-01-02"),
				"due_time":  startTime.Format("15:04:05"),
			}
			
			resp, err := p.makePipedriveRequest("POST", "/activities", activityData)
			if err != nil {
				return fmt.Errorf("failed to create call activity: %v", err)
			}
			defer resp.Body.Close()
			
			log.Printf("✅ Created call analyzed activity for unknown contact")
			return nil
		}
		
		log.Printf("📝 Found call mapping: %s (%s) - %s", callMapping.PersonName, callMapping.PhoneNumber, callMapping.LeadTitle)
		
		// Use stored person ID
		personID := callMapping.PersonID

		// Update person with call data in custom fields
		if err := p.UpdatePersonWithCallData(personID, payload.Call.Transcript, duration, startTime.Format("2006-01-02")); err != nil {
			log.Printf("⚠️ Warning: Failed to update person with call data: %v", err)
			// Continue with activity creation even if person update fails
		}

		// Create comprehensive call activity with person name
		activityData := map[string]interface{}{
			"subject":   fmt.Sprintf("AI Call Analyzed - %s", payload.Call.AgentName),
			"type":      "call",
			"person_id": personID,
			"duration":  duration,
			"note":      p.buildCallAnalyzedNoteWithPerson(payload, startTime, endTime, duration, callMapping.PersonName, callMapping.LeadTitle, callMapping.PhoneNumber),
			"done":      1,
			"due_date":  startTime.Format("2006-01-02"),
			"due_time":  startTime.Format("15:04:05"),
		}

		log.Printf("🔧 [DEBUG] Activity data: %+v", activityData)

		resp, err := p.makePipedriveRequest("POST", "/activities", activityData)
		if err != nil {
			log.Printf("❌ [DEBUG] Error creating activity: %v", err)
			return fmt.Errorf("failed to create call activity: %v", err)
		}
		defer resp.Body.Close()

		log.Printf("🔧 [DEBUG] Activity creation response status: %d", resp.StatusCode)

		var activityResult PipedriveActivityResponse
		if err := json.NewDecoder(resp.Body).Decode(&activityResult); err != nil {
			log.Printf("❌ [DEBUG] Error decoding activity response: %v", err)
			return fmt.Errorf("failed to decode activity response: %v", err)
		}

		log.Printf("🔧 [DEBUG] Activity result: %+v", activityResult)

		if !activityResult.Success {
			return fmt.Errorf("failed to create call activity in Pipedrive")
		}

		log.Printf("✅ Created call analyzed activity in Pipedrive: ID=%d", activityResult.Data.ID)

		// Add transcript as a note
		noteData := map[string]interface{}{
			"content":   fmt.Sprintf("Call Analysis:\n\n%s\n\nFull Transcript:\n%s", payload.Call.CallAnalysis.CallSummary, payload.Call.Transcript),
			"person_id": personID,
		}

		noteResp, err := p.makePipedriveRequest("POST", "/notes", noteData)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to create transcript note: %v", err)
		} else {
			noteResp.Body.Close()
			log.Printf("✅ Added transcript note for contact %d", personID)
		}

	} else {
		log.Printf("🔍 [SIMULATION MODE] Processing Retell call_analyzed webhook")
		log.Printf("   Call ID: %s", payload.Call.CallID)
		log.Printf("   Agent: %s", payload.Call.AgentName)
		log.Printf("   Duration: %d ms", payload.Call.DurationMs)
		log.Printf("   Status: %s", payload.Call.CallStatus)
		log.Printf("   Disconnection: %s", payload.Call.DisconnectionReason)
		log.Printf("   Sentiment: %s", payload.Call.CallAnalysis.UserSentiment)
		log.Printf("   Successful: %t", payload.Call.CallAnalysis.CallSuccessful)
		log.Printf("   ⚠️  This is a SIMULATION SERVER - not real Retell AI or Pipedrive")
	}

	return nil
}

// buildCallAnalyzedNote creates a comprehensive note for call_analyzed events
func (p *PipedriveService) buildCallAnalyzedNote(payload RetellCallAnalyzedPayload, startTime, endTime time.Time, duration string) string {
	note := fmt.Sprintf(`AI Call Analysis Report

Call Details:
• Call ID: %s
• Agent: %s (v%d)
• Type: %s
• Status: %s
• Duration: %s
• Start: %s
• End: %s
• Disconnection: %s

Call Analysis:
• Summary: %s
• User Sentiment: %s
• Call Successful: %t
• In Voicemail: %t

Transcript:
%s

Additional Resources:
• Recording: %s
• Multi-Channel Recording: %s
• Public Log: %s`,
		payload.Call.CallID,
		payload.Call.AgentName,
		payload.Call.AgentVersion,
		payload.Call.CallType,
		payload.Call.CallStatus,
		duration,
		startTime.Format("Monday, January 2, 2006 at 3:04 PM"),
		endTime.Format("Monday, January 2, 2006 at 3:04 PM"),
		payload.Call.DisconnectionReason,
		payload.Call.CallAnalysis.CallSummary,
		payload.Call.CallAnalysis.UserSentiment,
		payload.Call.CallAnalysis.CallSuccessful,
		payload.Call.CallAnalysis.InVoicemail,
		payload.Call.Transcript,
		payload.Call.RecordingURL,
		payload.Call.RecordingMultiChannelURL,
		payload.Call.PublicLogURL)

	return note
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

// handleCallStarted handles when a call begins
func (p *PipedriveService) handleCallStarted(personID int, payload RetellWebhookPayload, callTime time.Time) error {
	log.Printf("🔧 [DEBUG] Starting handleCallStarted for personID: %d", personID)
	
	// Create activity for call started
	activityData := map[string]interface{}{
		"subject":   "AI Call Started",
		"type":      "call",
		"person_id": personID,
		"note":      fmt.Sprintf("Retell AI call started\nCall ID: %s\nPhone: %s\nStarted at: %s", 
			payload.CallID, payload.ContactPhone, callTime.Format("2006-01-02 15:04:05")),
		"done":      0, // Mark as pending
		"due_date":  callTime.Format("2006-01-02"),
		"due_time":  callTime.Format("15:04:05"),
	}

	resp, err := p.makePipedriveRequest("POST", "/activities", activityData)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to create call started activity: %v", err)
	} else {
		resp.Body.Close()
		log.Printf("✅ Created call started activity for person %d", personID)
	}

	return nil
}

// handleCallEnded handles when a call ends (comprehensive end event)
func (p *PipedriveService) handleCallEnded(personID int, payload RetellWebhookPayload, callTime time.Time) error {
	log.Printf("🔧 [DEBUG] Starting handleCallEnded for personID: %d", personID)
	
	// Create activity for call ended
	activityData := map[string]interface{}{
		"subject":   "AI Call Ended",
		"type":      "call",
		"person_id": personID,
		"note":      fmt.Sprintf("Retell AI call ended\nCall ID: %s\nPhone: %s\nDuration: %s\nStatus: %s\nEnded at: %s", 
			payload.CallID, payload.ContactPhone, payload.Duration, payload.Status, callTime.Format("2006-01-02 15:04:05")),
		"done":      1, // Mark as completed
		"due_date":  callTime.Format("2006-01-02"),
		"due_time":  callTime.Format("15:04:05"),
	}

	resp, err := p.makePipedriveRequest("POST", "/activities", activityData)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to create call ended activity: %v", err)
	} else {
		resp.Body.Close()
		log.Printf("✅ Created call ended activity for person %d", personID)
	}

	return nil
}

// handleCallCompleted handles completed calls
func (p *PipedriveService) handleCallCompleted(personID int, payload RetellWebhookPayload, callTime time.Time) error {
	log.Printf("🔧 [DEBUG] Starting handleCallCompleted for personID: %d", personID)
	
	// Update person with call data in custom fields
	if err := p.UpdatePersonWithCallData(personID, payload.Transcript, payload.Duration, callTime.Format("2006-01-02")); err != nil {
		log.Printf("⚠️ Warning: Failed to update person with call data: %v", err)
		// Continue with activity creation even if person update fails
	}

	// Create call activity
	activityData := map[string]interface{}{
		"subject":   "AI Call Completed",
		"type":      "call",
		"person_id": personID,
		"duration":  payload.Duration,
		"note":      fmt.Sprintf("AI Call Completed\n\nCall ID: %s\nPhone: %s\nDuration: %s\nDate: %s\nTime: %s\nStatus: %s\nEvent: %s\n\nTranscript:\n%s", payload.CallID, payload.ContactPhone, payload.Duration, callTime.Format("Monday, January 2, 2006"), callTime.Format("3:04 PM"), payload.Status, payload.Event, payload.Transcript),
		"done":      1,
		"due_date":  callTime.Format("2006-01-02"),
		"due_time":  callTime.Format("15:04:05"),
	}
	
	log.Printf("🔧 [DEBUG] Activity data: %+v", activityData)
	
	resp, err := p.makePipedriveRequest("POST", "/activities", activityData)
	if err != nil {
		log.Printf("❌ [DEBUG] Error creating activity: %v", err)
		return fmt.Errorf("failed to create call activity: %v", err)
	}
	defer resp.Body.Close()
	
	log.Printf("🔧 [DEBUG] Activity creation response status: %d", resp.StatusCode)
	
	var activityResult PipedriveActivityResponse
	if err := json.NewDecoder(resp.Body).Decode(&activityResult); err != nil {
		log.Printf("❌ [DEBUG] Error decoding activity response: %v", err)
		return fmt.Errorf("failed to decode activity response: %v", err)
	}
	
	log.Printf("🔧 [DEBUG] Activity result: %+v", activityResult)
	
	if !activityResult.Success {
		log.Printf("❌ [DEBUG] Activity creation failed in Pipedrive")
		return fmt.Errorf("failed to create call activity in Pipedrive")
	}
	
	log.Printf("✅ Created call activity in Pipedrive: ID=%d", activityResult.Data.ID)

	log.Printf("🔧 [DEBUG] handleCallCompleted completed successfully")
	return nil
}

// handleCallHangup handles customer hang-ups
func (p *PipedriveService) handleCallHangup(personID int, payload RetellWebhookPayload, callTime time.Time) error {
	// Update person with call data in custom fields
	if err := p.UpdatePersonWithCallData(personID, payload.Transcript, payload.Duration, callTime.Format("2006-01-02")); err != nil {
		log.Printf("⚠️ Warning: Failed to update person with call data: %v", err)
		// Continue with activity creation even if person update fails
	}

	hangupData := map[string]interface{}{
		"subject":   "Customer Hung Up",
		"type":      "call",
		"person_id": personID,
		"note":      fmt.Sprintf("Customer Hung Up\n\nCall ID: %s\nPhone: %s\nDuration: %s\nDate: %s\nTime: %s\nStatus: %s\nEvent: %s\n\nTranscript:\n%s", payload.CallID, payload.ContactPhone, payload.Duration, callTime.Format("Monday, January 2, 2006"), callTime.Format("3:04 PM"), payload.Status, payload.Event, payload.Transcript),
		"done":      1,
		"due_date":  callTime.Format("2006-01-02"),
		"due_time":  callTime.Format("15:04:05"),
	}
	
	resp, err := p.makePipedriveRequest("POST", "/activities", hangupData)
	if err != nil {
		return fmt.Errorf("failed to create hangup activity: %v", err)
	}
	defer resp.Body.Close()
	
	var hangupResult PipedriveActivityResponse
	if err := json.NewDecoder(resp.Body).Decode(&hangupResult); err != nil {
		return fmt.Errorf("failed to decode hangup response: %v", err)
	}
	
	if hangupResult.Success {
		log.Printf("✅ Created hangup activity in Pipedrive: ID=%d", hangupResult.Data.ID)
	}
	
	return nil
}

// handleCallOptout handles opt-out requests
func (p *PipedriveService) handleCallOptout(personID int, payload RetellWebhookPayload, callTime time.Time) error {
	// Update person with call data in custom fields
	if err := p.UpdatePersonWithCallData(personID, payload.Transcript, payload.Duration, callTime.Format("2006-01-02")); err != nil {
		log.Printf("⚠️ Warning: Failed to update person with call data: %v", err)
		// Continue with other operations even if person update fails
	}

	// Update contact with DNC label
	updateData := map[string]interface{}{
		"label": "Do Not Contact",
	}
	
	endpoint := fmt.Sprintf("/persons/%d", personID)
	resp, err := p.makePipedriveRequest("PUT", endpoint, updateData)
	if err != nil {
		return fmt.Errorf("failed to mark as DNC: %v", err)
	}
	defer resp.Body.Close()
	
	log.Printf("✅ Marked contact %d as Do Not Contact (DNC)", personID)
	
	// Also create an activity for the opt-out
	optoutData := map[string]interface{}{
		"subject":   "Customer Opted Out",
		"type":      "call",
		"person_id": personID,
		"note":      fmt.Sprintf("Customer Opted Out\n\nCall ID: %s\nPhone: %s\nDuration: %s\nDate: %s\nTime: %s\nStatus: %s\nEvent: %s\n\nTranscript:\n%s\n\nCustomer requested to be removed from contact list.", payload.CallID, payload.ContactPhone, payload.Duration, callTime.Format("Monday, January 2, 2006"), callTime.Format("3:04 PM"), payload.Status, payload.Event, payload.Transcript),
		"done":      1,
		"due_date":  callTime.Format("2006-01-02"),
		"due_time":  callTime.Format("15:04:05"),
	}
	
	resp, err = p.makePipedriveRequest("POST", "/activities", optoutData)
	if err != nil {
		return fmt.Errorf("failed to create optout activity: %v", err)
	}
	defer resp.Body.Close()
	
	return nil
}

// addTranscriptNote adds transcript as a note to the contact
func (p *PipedriveService) addTranscriptNote(personID int, transcript string) error {
	noteData := map[string]interface{}{
		"content":   fmt.Sprintf("Transcript:\n%s", transcript),
		"person_id": personID,
	}
	
	resp, err := p.makePipedriveRequest("POST", "/notes", noteData)
	if err != nil {
		return fmt.Errorf("failed to create transcript note: %v", err)
	}
	defer resp.Body.Close()
	
	log.Printf("✅ Added transcript note for contact %d", personID)
	return nil
}

// FindOrCreateContactByPhone finds or creates a contact by phone number
func (p *PipedriveService) FindOrCreateContactByPhone(phone string) (*Contact, error) {
	if p.config.HasPipedriveConfig() {
		log.Printf("🔍 [REAL PIPEDRIVE API] Searching for contact by phone: %s", phone)
		
		// Search for existing contact by phone
		searchEndpoint := fmt.Sprintf("/persons/search?term=%s&fields=phone", phone)
		resp, err := p.makePipedriveRequest("GET", searchEndpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to search contact: %v", err)
		}
		defer resp.Body.Close()
		
		var searchResult PipedrivePersonSearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
			return nil, fmt.Errorf("failed to decode search response: %v", err)
		}
		
		// If contact found, return it
		if searchResult.Success && len(searchResult.Items) > 0 {
			person := searchResult.Items[0]
			phone := ""
			email := ""
			if len(person.Phone) > 0 {
				phone = person.Phone[0].Value
			}
			if len(person.Email) > 0 {
				email = person.Email[0].Value
			}
			log.Printf("✅ Found existing contact in Pipedrive: ID=%d, Name=%s", person.ID, person.Name)
			return &Contact{
				ID:    fmt.Sprintf("%d", person.ID),
				Name:  person.Name,
				Email: email,
				Phone: phone,
				DNC:   false,
			}, nil
		}
		
		// Create new contact if not found
		log.Printf("📝 Creating new contact in Pipedrive for phone: %s", phone)
		createData := map[string]interface{}{
			"name":  "Unknown Caller",
			"phone": []map[string]string{{"value": phone}},
		}
		
		resp, err = p.makePipedriveRequest("POST", "/persons", createData)
		if err != nil {
			return nil, fmt.Errorf("failed to create contact: %v", err)
		}
		defer resp.Body.Close()
		
		var createResult PipedrivePersonResponse
		if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
			return nil, fmt.Errorf("failed to decode create response: %v", err)
		}
		
		if !createResult.Success || createResult.Data == nil {
			return nil, fmt.Errorf("failed to create contact in Pipedrive")
		}
		
		person := createResult.Data
		phone := ""
		email := ""
		if len(person.Phone) > 0 {
			phone = person.Phone[0].Value
		}
		if len(person.Email) > 0 {
			email = person.Email[0].Value
		}
		log.Printf("✅ Created new contact in Pipedrive: ID=%d, Name=%s", person.ID, person.Name)
		return &Contact{
			ID:    fmt.Sprintf("%d", person.ID),
			Name:  person.Name,
			Email: email,
			Phone: phone,
			DNC:   false,
		}, nil
		
	} else {
		// Simulation mode
		log.Printf("🔍 [SIMULATION MODE] Searching for contact by phone: %s", phone)
		contact := &Contact{
			ID:    uuid.New().String(),
			Name:  "Unknown Caller",
			Email: "",
			Phone: phone,
			DNC:   false,
		}
		
		log.Printf("✅ Contact found/created: ID=%s, Phone=%s", contact.ID, contact.Phone)
		return contact, nil
	}
}

// FindOrCreateContactByEmail finds or creates a contact by email address
func (p *PipedriveService) FindOrCreateContactByEmail(email, name string) (*Contact, error) {
	log.Printf("🔍 [REAL PIPEDRIVE API] Searching for contact by email: %s", email)

	// Search for existing contact by email
	// URL-encode the email to handle special characters like @ and +
	encodedEmail := url.QueryEscape(email)
	searchEndpoint := fmt.Sprintf("/persons/search?term=%s&fields=email", encodedEmail)
	resp, err := p.makePipedriveRequest("GET", searchEndpoint, nil)
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

// FindLeadByEmail searches for existing leads in Pipedrive by email
func (p *PipedriveService) FindLeadByEmail(email string) (*PipedriveLead, error) {
	if !p.config.HasPipedriveConfig() {
		return nil, fmt.Errorf("Pipedrive not configured")
	}

	log.Printf("🔍 [REAL PIPEDRIVE API] Searching for leads by email: %s", email)

	// First, find the person by email
	person, err := p.FindOrCreateContactByEmail(email, "")
	if err != nil {
		return nil, fmt.Errorf("failed to find person by email: %v", err)
	}

	personID, err := strconv.Atoi(person.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid person ID: %v", err)
	}

	// Search for leads associated with this person
	searchURL := fmt.Sprintf("/leads?person_id=%d", personID)
	resp, err := p.makePipedriveRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for leads: %v", err)
	}
	defer resp.Body.Close()

	var leadResult PipedriveLeadSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&leadResult); err != nil {
		return nil, fmt.Errorf("failed to decode lead search response: %v", err)
	}

	// If leads found, return the first one
	if leadResult.Success && len(leadResult.Items) > 0 {
		lead := leadResult.Items[0]
		log.Printf("✅ Found existing lead: ID=%s, Title=%s", lead.ID, lead.Title)
		return &lead, nil
	}

	log.Printf("ℹ️ No leads found for person ID: %d", personID)
	return nil, nil
}

// UpdatePersonWithCallData updates a person with call data in custom fields
func (p *PipedriveService) UpdatePersonWithCallData(personID int, transcript, duration, date string) error {
	updateData := map[string]interface{}{
		"b4073939104c3d1283e703c3b3e9fb261a16b137": transcript, // transcript field
		"22d4bfd3fc0227ef6f8a594346c30545b069d5fd": duration,   // call_duration field
		"80347870cd9400fbc1a1d03bd082df463321bad5": date,       // date_call field
	}

	log.Printf("🔧 [DEBUG] Updating person %d with call data", personID)

	endpoint := fmt.Sprintf("/persons/%d", personID)
	resp, err := p.makePipedriveRequest("PUT", endpoint, updateData)
	if err != nil {
		log.Printf("❌ [DEBUG] Error making update request: %v", err)
		return fmt.Errorf("failed to update person with call data: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("🔧 [DEBUG] Update response status: %d", resp.StatusCode)

	// Check if the update was successful
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to update person: HTTP %d", resp.StatusCode)
	}

	log.Printf("✅ Updated person %d with custom fields", personID)
	return nil
}

// ProcessCalAppointment processes a Cal.com appointment webhook
func (p *PipedriveService) ProcessCalAppointment(payload CalWebhookPayload) error {
	log.Printf("🔧 [DEBUG] ProcessCalAppointment called")
	log.Printf("🔧 [DEBUG] HasPipedriveConfig: %v", p.config.HasPipedriveConfig())
	log.Printf("🔧 [DEBUG] PIPEDRIVE_API_KEY: %s", p.config.PipedriveAPIKey)

	if p.config.HasPipedriveConfig() {
		log.Printf("🚀 [REAL PIPEDRIVE] Processing Cal.com appointment webhook")

		// Parse start and end times
		startTime, err := time.Parse(time.RFC3339, payload.Payload.StartTime)
		if err != nil {
			log.Printf("❌ [DEBUG] Error parsing startTime: %v", err)
			return fmt.Errorf("invalid startTime format: %v", err)
		}

		endTime, err := time.Parse(time.RFC3339, payload.Payload.EndTime)
		if err != nil {
			log.Printf("❌ [DEBUG] Error parsing endTime: %v", err)
			return fmt.Errorf("invalid endTime format: %v", err)
		}

		// Calculate duration
		duration := endTime.Sub(startTime)
		durationStr := fmt.Sprintf("%02d:%02d:%02d", 
			int(duration.Hours()), 
			int(duration.Minutes())%60, 
			int(duration.Seconds())%60)

		// Get the first attendee (main contact)
		attendee := payload.Payload.Attendees[0]
		log.Printf("📧 [DEBUG] Processing attendee: %s (%s)", attendee.Name, attendee.Email)

		// First, search for existing leads by email
		lead, err := p.FindLeadByEmail(attendee.Email)
		if err != nil {
			log.Printf("⚠️ [DEBUG] Error searching for leads: %v", err)
			// Continue with contact creation even if lead search fails
		}

		var personID int
		var personName string

		if lead != nil {
			// Lead found, use the existing person
			personID = lead.PersonID
			log.Printf("✅ [DEBUG] Found existing lead: ID=%s, Title=%s, PersonID=%d", lead.ID, lead.Title, lead.PersonID)
			
			// Get person details
			person, err := p.FindOrCreateContactByEmail(attendee.Email, attendee.Name)
			if err != nil {
				log.Printf("⚠️ [DEBUG] Error getting person details: %v", err)
				personName = attendee.Name
			} else {
				personName = person.Name
			}
		} else {
			// No lead found, create new contact
			log.Printf("ℹ️ [DEBUG] No existing lead found, creating new contact")
			contact, err := p.FindOrCreateContactByEmail(attendee.Email, attendee.Name)
			if err != nil {
				log.Printf("❌ [DEBUG] Error finding/creating contact: %v", err)
				return fmt.Errorf("failed to find/create contact: %v", err)
			}

			personID, err = strconv.Atoi(contact.ID)
			if err != nil {
				log.Printf("❌ [DEBUG] Error converting contact ID: %v", err)
				return fmt.Errorf("invalid contact ID: %v", err)
			}
			personName = contact.Name
		}

		log.Printf("✅ [DEBUG] Using person: ID=%d, Name=%s", personID, personName)

		// Create detailed appointment activity note
		note := p.buildCalAppointmentNote(payload, startTime, endTime, durationStr, personName, attendee)

		// Create appointment activity in Pipedrive
		activityData := map[string]interface{}{
			"subject":   fmt.Sprintf("📅 Cal.com: %s", payload.Payload.Title),
			"type":      "meeting",
			"person_id": personID,
			"note":      note,
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

// buildCalAppointmentNote creates a detailed note for Cal.com appointments
func (p *PipedriveService) buildCalAppointmentNote(payload CalWebhookPayload, startTime, endTime time.Time, duration, personName string, attendee struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}) string {
	// Format times for display
	startTimeStr := startTime.Format("Monday, January 2, 2006 at 3:04 PM")
	endTimeStr := endTime.Format("Monday, January 2, 2006 at 3:04 PM")
	dateStr := startTime.Format("2006-01-02")
	
	// Create detailed note with all appointment information
	note := fmt.Sprintf(`📅 Cal.com Appointment Scheduled

👤 Person: %s
📧 Email: %s
📞 Attendee: %s

📋 Appointment Details:
• Title: %s
• Booking ID: %d
• Date: %s
• Start Time: %s
• End Time: %s
• Duration: %s
• Location: %s

🔗 Meeting Information:
• Meeting URL: %s
• Event Type: %s
• Created: %s

📝 Additional Attendees:`,
		personName,
		attendee.Email,
		attendee.Name,
		payload.Payload.Title,
		payload.Payload.ID,
		dateStr,
		startTimeStr,
		endTimeStr,
		duration,
		payload.Payload.Location,
		payload.Payload.Location,
		payload.TriggerEvent,
		payload.CreatedAt)

	// Add all attendees
	for i, att := range payload.Payload.Attendees {
		note += fmt.Sprintf("\n  %d. %s (%s)", i+1, att.Name, att.Email)
	}

	note += fmt.Sprintf(`

📊 Summary:
This appointment was automatically created from Cal.com webhook. The meeting is scheduled for %s and will last %s.`, 
		startTimeStr, duration)

	return note
}

// extractPhoneFromPerson extracts phone from PipedrivePerson
func extractPhoneFromPerson(person *PipedrivePerson) string {
	if len(person.Phone) > 0 {
		return person.Phone[0].Value
	}
	return ""
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

// getCallMapping retrieves call information by call ID
func (p *PipedriveService) getCallMapping(callID string) (CallMapping, bool) {
	mapping, exists := p.callMappings[callID]
	return mapping, exists
}

// buildCallAnalyzedNoteWithPerson creates a comprehensive note for call analysis with person details
func (p *PipedriveService) buildCallAnalyzedNoteWithPerson(payload RetellCallAnalyzedPayload, startTime, endTime time.Time, duration, personName, leadTitle, phoneNumber string) string {
	return fmt.Sprintf(`🤖 AI Call Analysis Complete

👤 Person: %s
📞 Phone: %s
🎯 Lead: %s
📅 Date: %s
⏰ Time: %s - %s
⏱️ Duration: %s

📊 Analysis Summary:
%s

😊 Sentiment: %s
✅ Call Successful: %t
📝 Disconnection Reason: %s

🤖 Agent: %s (v%d)
📋 Call ID: %s

📄 Full Transcript:
%s`, 
		personName,
		phoneNumber,
		leadTitle,
		startTime.Format("2006-01-02"),
		startTime.Format("15:04:05"),
		endTime.Format("15:04:05"),
		duration,
		payload.Call.CallAnalysis.CallSummary,
		payload.Call.CallAnalysis.UserSentiment,
		payload.Call.CallAnalysis.CallSuccessful,
		payload.Call.DisconnectionReason,
		payload.Call.AgentName,
		payload.Call.AgentVersion,
		payload.Call.CallID,
		payload.Call.Transcript)
}
