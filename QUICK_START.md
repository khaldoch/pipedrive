# PipCal - Quick Start Guide

## TL;DR - What Was Fixed

1. ✅ **Cleaned up messy file structure** - Removed duplicate files
2. ✅ **Fixed Retell AI integration** - Was always there, just had wrong URLs
3. ✅ **Fixed Vercel deployment** - Removed wrong `@` syntax, added `.vercelignore`

---

## Local Development (Fast)

```bash
# 1. Run the server
go run .

# 2. Test it works
curl http://localhost:8080/health

# 3. Test Pipedrive lead webhook
curl -X POST http://localhost:8080/test/pipedrive-lead
```

**Your `.env` is already configured!** ✅

---

## Deploy to Vercel (3 Steps)

### Step 1: Set Environment Variables in Vercel Dashboard

Go to: **Vercel Dashboard → Your Project → Settings → Environment Variables**

Add these:

| Key | Value |
|-----|-------|
| `PIPEDRIVE_API_KEY` | `fc235b34f32fb79eb0b17637a40d64b8f8d1234d` |
| `PIPEDRIVE_BASE_URL` | `https://api.pipedrive.com/v1` |
| `RETELL_API_KEY` | `key_76c72b1e7aaf38586b9d5ff3ab2f` |
| `RETELL_ASSISTANT_ID` | `agent_f253ac7638891811859d98923e` |
| `RETELL_FROM_NUMBER` | `18005300627` |
| `RETELL_BASE_URL` | `https://api.retellai.com` |
| `GIN_MODE` | `release` |

### Step 2: Deploy

```bash
# Option A: Use the script
./deploy-vercel-fixed.sh

# Option B: Manual
vercel --prod
```

### Step 3: Configure Webhooks

**Pipedrive:**
- URL: `https://your-project.vercel.app/webhook/pipedrive/lead`
- Event: **Lead - created**

**Retell AI:**
- URL: `https://your-project.vercel.app/webhook/retell/analyzed`
- Events: `call_analyzed`, `call_started`, `call_ended`

---

## Test Deployment

```bash
# Health check
curl https://your-project.vercel.app/health

# Test Pipedrive webhook
curl -X POST https://your-project.vercel.app/test/pipedrive-lead
```

---

## How It Works

1. **New lead created in Pipedrive** → Webhook fires
2. **Server gets person details** → Fetches from Pipedrive API
3. **Server creates Retell AI call** → `POST /v2/create-phone-call`
4. **Retell AI calls the lead** → AI assistant talks to them
5. **Results logged back** → Activity created in Pipedrive

---

## Files to Know

| File | Purpose |
|------|---------|
| `main.go` | Local server (for development) |
| `api/` | Vercel serverless functions |
| `.env` | Your API keys (already configured) |
| `vercel.json` | Vercel config (fixed) |
| `.vercelignore` | Prevents deploying root files |

---

## Troubleshooting

### "Port 8080 already in use"
```bash
pkill pipcal-server
go run .
```

### "Vercel build failed"
Check that environment variables are set in Vercel Dashboard (not in `vercel.json`)

### "Retell AI 404 error"
Fixed! URLs are now correct:
- ✅ Base: `https://api.retellai.com`
- ✅ Endpoint: `/v2/create-phone-call`

---

## Documentation

- **Complete setup:** `SETUP.md`
- **What was fixed:** `FIXES_APPLIED.md`
- **Vercel deployment:** `VERCEL_DEPLOYMENT.md`
- **Vercel fixes:** `VERCEL_FIXES_SUMMARY.md`

---

## Summary

**Before:** Messy structure, wrong URLs, Retell AI not working, Vercel failing

**After:** Clean structure, correct URLs, Retell AI working, Vercel ready to deploy

**Ready to go!** 🚀
