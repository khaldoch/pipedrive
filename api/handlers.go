package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RetellWebhookHandler handles Retell AI webhook requests
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

// RetellCallAnalyzedHandler handles Retell AI call_analyzed webhook requests
func RetellCallAnalyzedHandler(pipedriveService *PipedriveService) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("🔔 [WEBHOOK] Received Retell call_analyzed webhook")

		var payload RetellCallAnalyzedPayload

		// Bind JSON payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			log.Printf("❌ [WEBHOOK ERROR] Invalid JSON payload: %v", err)
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Invalid JSON payload",
			})
			return
		}

		log.Printf("📦 [WEBHOOK] Received call_analyzed for Call ID: %s", payload.Call.CallID)
		log.Printf("📦 [WEBHOOK] Event type: %s", payload.Event)
		log.Printf("📦 [WEBHOOK] Agent: %s", payload.Call.AgentName)
		log.Printf("📦 [WEBHOOK] Duration: %d ms", payload.Call.DurationMs)
		log.Printf("📦 [WEBHOOK] Status: %s", payload.Call.CallStatus)
		log.Printf("📦 [WEBHOOK] Transcript length: %d chars", len(payload.Call.Transcript))

		// Validate required fields
		if payload.Call.CallID == "" {
			log.Printf("❌ [WEBHOOK ERROR] Missing call_id in payload")
			c.JSON(http.StatusBadRequest, WebhookResponse{
				Success: false,
				Message: "Missing required field: call.call_id",
			})
			return
		}

		log.Printf("🔄 [WEBHOOK] Processing call_analyzed webhook...")

		// Process the call analyzed
		if err := pipedriveService.ProcessRetellCallAnalyzed(payload); err != nil {
			log.Printf("❌ [WEBHOOK ERROR] Failed to process: %v", err)
			c.JSON(http.StatusInternalServerError, WebhookResponse{
				Success: false,
				Message: "Failed to process call analyzed: " + err.Error(),
			})
			return
		}

		log.Printf("✅ [WEBHOOK] Successfully processed call_analyzed webhook for Call ID: %s", payload.Call.CallID)

		// Return success response
		c.JSON(http.StatusOK, WebhookResponse{
			Success: true,
			Message: "Retell call_analyzed webhook processed successfully",
			Data: gin.H{
				"call_id":    payload.Call.CallID,
				"agent_name": payload.Call.AgentName,
				"duration":   payload.Call.DurationMs,
				"status":     payload.Call.CallStatus,
				"sentiment":  payload.Call.CallAnalysis.UserSentiment,
			},
		})
	}
}

// PipedriveLeadWebhookHandler handles Pipedrive lead webhook requests
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
				"lead_id":   payload.Data.ID,
				"person_id": payload.Data.PersonID,
				"title":     payload.Data.Title,
				"action":    payload.Meta.Action,
			},
		})
	}
}

// CalWebhookHandler handles Cal.com webhook requests
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

// HealthCheckHandler provides a simple health check endpoint
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "PipCal Webhook Server",
		"version": "1.0.0",
	})
}
