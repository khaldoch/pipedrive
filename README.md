# PipCal Webhook Server

A Go-based webhook server that simulates Retell AI and Cal.com integrations with Pipedrive. This server processes incoming webhook data and simulates the workflow of logging activities and managing contacts in Pipedrive.

## Features

- **Retell AI Webhook Simulation**: Processes call data including transcripts, duration, and contact information
- **Cal.com Webhook Simulation**: Handles appointment booking data with meeting URLs
- **Pipedrive Integration Simulation**: Simulates contact lookup/creation and activity logging
- **Comprehensive Logging**: Detailed console output showing the simulated workflow
- **Postman Collection**: Ready-to-use API testing collection

## Project Structure

```
pipcal/
├── main.go                              # Main server entry point
├── models.go                            # Data structures and models
├── services.go                          # Pipedrive simulation logic
├── handlers.go                          # HTTP request handlers
├── go.mod                              # Go module dependencies
├── PipCal_Webhooks.postman_collection.json  # Postman collection
└── README.md                           # This file
```

## Prerequisites

- Go 1.21 or higher
- Postman (for testing) or any HTTP client

## Installation & Setup

1. **Clone or download the project files**

2. **Install dependencies:**
   ```bash
   go mod tidy
   ```

3. **Configure environment variables (optional):**
   ```bash
   # Copy the example environment file
   cp env.example .env
   
   # Edit .env with your configuration
   nano .env
   ```

4. **Run the server:**
   ```bash
   # Basic run (simulation mode)
   go run .
   
   # With custom port
   PORT=3000 go run .
   
   # With Pipedrive API (real integration)
   PIPEDRIVE_API_KEY=your_api_key go run .
   ```

   The server will start on `http://localhost:8080` by default.

## API Endpoints

### Health Check
- **GET** `/health` - Check server status

### Webhooks
- **POST** `/webhook/retell` - Retell AI call webhook
- **POST** `/webhook/cal` - Cal.com appointment webhook

## Testing with Postman

1. **Import the Collection:**
   - Open Postman
   - Click "Import" and select `PipCal_Webhooks.postman_collection.json`
   - The collection will be imported with all example requests

2. **Set Environment Variable:**
   - The collection uses `{{base_url}}` variable set to `http://localhost:8080`
   - Modify if your server runs on a different port

3. **Run Tests:**
   - Start the server: `go run .`
   - Run any request from the collection
   - Check the server console for detailed logs

## Example Requests

### Retell AI Webhook (Call Completed)
```bash
curl -X POST http://localhost:8080/webhook/retell \
  -H "Content-Type: application/json" \
  -d '{
    "contact": {
      "name": "John Doe",
      "email": "john.doe@example.com",
      "phone": "+1-555-0123"
    },
    "transcript": "Hello, this is John calling about the product demo...",
    "duration": 5,
    "datetime": "2024-01-15T14:30:00Z",
    "hang_up": true,
    "opt_out": false
  }'
```

### Cal.com Webhook (Appointment Booked)
```bash
curl -X POST http://localhost:8080/webhook/cal \
  -H "Content-Type: application/json" \
  -d '{
    "contact": {
      "name": "Alice Brown",
      "email": "alice.brown@example.com",
      "phone": "+1-555-0321"
    },
    "start_time": "2024-01-20T09:00:00Z",
    "end_time": "2024-01-20T10:00:00Z",
    "meeting_url": "https://cal.com/meeting/abc123"
  }'
```

## Webhook Payloads

### Retell AI Payload Structure
```json
{
  "contact": {
    "name": "string",
    "email": "string",
    "phone": "string"
  },
  "transcript": "string",
  "duration": "integer (minutes)",
  "datetime": "string (ISO8601)",
  "hang_up": "boolean",
  "opt_out": "boolean"
}
```

### Cal.com Payload Structure
```json
{
  "contact": {
    "name": "string",
    "email": "string",
    "phone": "string"
  },
  "start_time": "string (ISO8601)",
  "end_time": "string (ISO8601)",
  "meeting_url": "string"
}
```

## Server Response Format

All webhook endpoints return JSON responses:

```json
{
  "success": "boolean",
  "message": "string",
  "data": "object (optional)"
}
```

## Environment Variables

The server supports the following environment variables:

### Server Configuration
- `PORT` - Server port (default: 8080)
- `HOST` - Server host (default: 0.0.0.0)
- `LOG_LEVEL` - Logging level (default: info)
- `GIN_MODE` - Gin framework mode (debug/release)

### Pipedrive API Configuration
- `PIPEDRIVE_API_KEY` - Your Pipedrive API key
- `PIPEDRIVE_BASE_URL` - Pipedrive API base URL (default: https://api.pipedrive.com/v1)
- `PIPEDRIVE_COMPANY_ID` - Your Pipedrive company ID

### Webhook Security (Optional)
- `RETELL_WEBHOOK_SECRET` - Secret for Retell webhook verification
- `CAL_WEBHOOK_SECRET` - Secret for Cal.com webhook verification

### Example .env file:
```bash
PORT=8080
PIPEDRIVE_API_KEY=your_api_key_here
PIPEDRIVE_BASE_URL=https://api.pipedrive.com/v1
PIPEDRIVE_COMPANY_ID=your_company_id
LOG_LEVEL=info
```

## Pipedrive Integration Notes

This is a **simulation** of Pipedrive integration by default. To enable real Pipedrive integration:

1. **Set environment variables** for Pipedrive API configuration
2. **Replace simulation methods** in `services.go` with actual Pipedrive API calls
3. **Add authentication** using the configured API tokens
4. **Implement error handling** for API failures
5. **Add rate limiting** and retry logic
6. **Use proper HTTP client** with timeouts and connection pooling

### Key Integration Points

- `FindOrCreateContact()` - Search/create contacts in Pipedrive
- `LogActivity()` - Create activity records
- `MarkContactAsDNC()` - Update contact DNC status

### Simulation vs Real API Mode

- **Simulation Mode** (default): Logs show `[SIMULATION]` and no API calls are made
- **Real API Mode**: Set `PIPEDRIVE_API_KEY` to enable real Pipedrive integration

## Development

### Adding New Webhook Types

1. Create new payload structure in `models.go`
2. Add handler function in `handlers.go`
3. Implement processing logic in `services.go`
4. Register route in `main.go`

### Building for Production

```bash
# Build binary
go build -o pipcal-server .

# Run binary
./pipcal-server
```

## Logging

The server provides detailed console logging showing:
- Contact lookup/creation
- Activity logging
- DNC marking
- Error conditions
- Request processing

## Error Handling

- Invalid JSON payloads return 400 Bad Request
- Missing required fields return 400 Bad Request
- Processing errors return 500 Internal Server Error
- All errors include descriptive messages

## License

This project is provided as-is for demonstration purposes.
# pipedrive
