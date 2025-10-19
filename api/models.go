package handler

import (
	"time"
)

// Contact represents a contact in the system
type Contact struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	DNC   bool   `json:"dnc"` // Do Not Call flag
}

// Activity represents an activity log entry
type Activity struct {
	ID          string    `json:"id"`
	ContactID   string    `json:"contact_id"`
	Type        string    `json:"type"` // "call", "appointment", "hang_up"
	Description string    `json:"description"`
	Duration    int       `json:"duration,omitempty"` // in minutes
	DateTime    time.Time `json:"datetime"`
	MeetingURL  string    `json:"meeting_url,omitempty"`
	Transcript  string    `json:"transcript,omitempty"`
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

// RetellCallAnalyzedPayload represents the call_analyzed webhook payload
type RetellCallAnalyzedPayload struct {
	Event string `json:"event"`
	Call  struct {
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
	} `json:"call"`
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

// ContactPayload represents contact data in webhook payloads
type ContactPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// WebhookResponse represents the response sent back to webhook callers
type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
