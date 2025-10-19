# PipCal - Pipedrive & Retell AI Integration

Complete webhook integration server that connects Pipedrive CRM with Retell AI for automated phone calling.

## Project Structure

```
pipCal/
├── api/                    # Vercel serverless functions
│   ├── index.go           # Main Vercel handler
│   ├── config.go          # Configuration management
│   ├── handlers.go        # Webhook handlers
│   ├── models.go          # Data models
│   └── services.go        # Business logic & API integrations
├── main.go                # Standalone server entry point
├── .env                   # Environment variables (DO NOT COMMIT)
├── vercel.json            # Vercel deployment configuration
└── README.md              # Documentation

## What It Does

### Workflow:

1. **New Lead in Pipedrive** → Triggers webhook to `/webhook/pipedrive/lead`
2. **Server fetches person details** from Pipedrive API (name, phone number)
3. **Server initiates Retell AI call** to the lead's phone number
4. **Retell AI makes the call** using your AI assistant
5. **Call results logged back** to Pipedrive as activities

## Fixed Issues

### Problems Solved:

1. ✅ **Messy file structure** - Removed duplicate files in root directory
2. ✅ **Wrong API URLs** - Fixed Pipedrive and Retell AI base URLs
3. ✅ **Missing Retell AI integration** - The code WAS there, just needed correct configuration
4. ✅ **Wrong Retell AI endpoint** - Changed from `/create-phone-call` to `/v2/create-phone-call`
5. ✅ **Configuration issues** - Fixed environment variable loading

## Environment Variables

Create a `.env` file with these variables:

```bash
# Server Configuration
PORT=8080
HOST=0.0.0.0
LOG_LEVEL=info
GIN_MODE=debug

# Pipedrive API Configuration
PIPEDRIVE_API_KEY=your_pipedrive_api_key_here
PIPEDRIVE_BASE_URL=https://api.pipedrive.com/v1
PIPEDRIVE_COMPANY_ID=your_company_id

# Retell AI Configuration
RETELL_API_KEY=your_retell_api_key_here
RETELL_ASSISTANT_ID=your_assistant_id_here
RETELL_FROM_NUMBER=+18005300627
RETELL_BASE_URL=https://api.retellai.com

# Webhook Security (Optional)
RETELL_WEBHOOK_SECRET=your_secret_here
CAL_WEBHOOK_SECRET=your_cal_secret_here
```

### How to Get API Keys:

#### Pipedrive API Key:
1. Log into Pipedrive
2. Go to Settings → Personal → API
3. Copy your API token

#### Retell AI API Key:
1. Log into https://retellai.com
2. Go to Dashboard → API Keys
3. Create a new API key
4. Copy the `assistant_id` from your AI assistant

## Installation & Setup

### Local Development:

```bash
# 1. Clone the repository
git clone <your-repo-url>
cd pipCal

# 2. Install dependencies
go mod tidy

# 3. Create .env file (see above)
cp .env.example .env
nano .env  # Edit with your API keys

# 4. Build and run
go build -o pipcal-server main.go
./pipcal-server

# Server will start on http://localhost:8080
```

### Vercel Deployment:

```bash
# 1. Install Vercel CLI
npm install -g vercel

# 2. Set environment variables in Vercel
vercel env add PIPEDRIVE_API_KEY
vercel env add RETELL_API_KEY
vercel env add RETELL_ASSISTANT_ID
vercel env add RETELL_FROM_NUMBER

# 3. Deploy
vercel --prod
```

## API Endpoints

### Webhooks:

- `POST /webhook/pipedrive/lead` - Pipedrive lead creation webhook
- `POST /webhook/retell` - Retell AI call events
- `POST /webhook/retell/analyzed` - Retell AI call analysis results
- `POST /webhook/cal` - Cal.com appointment bookings

### Test Endpoints:

- `POST /test/pipedrive-lead` - Test Pipedrive lead webhook
- `POST /test/completed` - Test Retell call completion
- `GET /health` - Health check

## Testing

### Test the Pipedrive Lead Webhook:

```bash
curl -X POST http://localhost:8080/test/pipedrive-lead \
  -H "Content-Type: application/json"
```

### Expected Behavior:

1. Server fetches person details from Pipedrive
2. Extracts phone number
3. Creates Retell AI call
4. Logs activity in Pipedrive

### Check Logs:

```bash
tail -f server.log
```

Look for:
- ✅ `[REAL INTEGRATION] Processing Pipedrive lead webhook`
- ✅ `Found phone number: +XXX for person: NAME`
- ✅ `Making Retell AI call to: https://api.retellai.com/v2/create-phone-call`
- ✅ `Successfully created Retell AI call: CALL_ID`

## Pipedrive Webhook Setup

1. Go to Pipedrive Settings → Webhooks
2. Create new webhook
3. Set URL to: `https://your-domain.com/webhook/pipedrive/lead`
4. Select event: `Lead - created`
5. Save

## Retell AI Webhook Setup

1. Go to Retell AI Dashboard → Webhooks
2. Add webhook URL: `https://your-domain.com/webhook/retell/analyzed`
3. Select events:
   - `call_analyzed`
   - `call_started`
   - `call_ended`

## Troubleshooting

### Issue: "Retell AI not configured"
**Solution:** Check that `RETELL_API_KEY` and `RETELL_ASSISTANT_ID` are set in `.env`

### Issue: "404 Cannot POST /create-phone-call"
**Solution:** Ensure `RETELL_BASE_URL=https://api.retellai.com` (not `/v2` suffix)

### Issue: "Pipedrive API not configured"
**Solution:** Set `PIPEDRIVE_API_KEY` in `.env`

### Issue: "Failed to get person details"
**Solution:** Verify the person ID exists in Pipedrive and API key has permissions

## Code Structure

### `main.go`
- Entry point for standalone server
- Configures Gin router
- Sets up all endpoints
- Includes both main() for standalone and Handler() for Vercel

### `api/services.go`
- `ProcessPipedriveLead()` - Main function that handles lead webhooks
- `CreateRetellCall()` - Creates phone calls via Retell AI API
- `GetPersonByID()` - Fetches person details from Pipedrive
- `storeCallMapping()` - Maps call IDs to person info for later use

### `api/handlers.go`
- HTTP handlers for all webhook endpoints
- Request validation
- Response formatting

### `api/models.go`
- Data structures for webhooks
- API request/response models

### `api/config.go`
- Environment variable loading
- Configuration validation

## Important Notes

1. **Phone Number Format**: Retell AI requires E.164 format (+COUNTRYCODE + NUMBER)
2. **Call Mapping**: The server stores a mapping between Retell call IDs and Pipedrive person IDs
3. **Activities**: All call attempts are logged as activities in Pipedrive
4. **Custom Fields**: Call data is stored in Pipedrive custom fields (transcript, duration, date)

## Security

- Never commit `.env` file
- Use webhook secrets in production
- Validate webhook signatures
- Use HTTPS in production

## Support

For issues or questions:
- Check logs: `tail -f server.log`
- Enable debug mode: `GIN_MODE=debug`
- Review Pipedrive API docs: https://developers.pipedrive.com
- Review Retell AI docs: https://docs.retellai.com
