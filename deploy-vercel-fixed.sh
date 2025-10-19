#!/bin/bash

# Vercel Deployment Script for PipCal
# This script helps you deploy to Vercel with proper configuration

echo "🚀 PipCal Vercel Deployment Helper"
echo "=================================="
echo ""

# Check if vercel CLI is installed
if ! command -v vercel &> /dev/null; then
    echo "❌ Vercel CLI not found!"
    echo "📦 Installing Vercel CLI..."
    npm install -g vercel
    echo ""
fi

# Check if user is logged in
echo "🔐 Checking Vercel login..."
if ! vercel whoami &> /dev/null; then
    echo "❌ Not logged in to Vercel"
    echo "🔑 Please login:"
    vercel login
    echo ""
fi

# Display current environment
echo "📋 Current configuration:"
echo "   PIPEDRIVE_API_KEY: ${PIPEDRIVE_API_KEY:-Not set in .env}"
echo "   RETELL_API_KEY: ${RETELL_API_KEY:-Not set in .env}"
echo "   RETELL_ASSISTANT_ID: ${RETELL_ASSISTANT_ID:-Not set in .env}"
echo ""

# Ask user if they want to set environment variables
echo "⚙️  Do you want to set environment variables in Vercel?"
echo "   (Choose 'yes' if this is your first deployment)"
read -p "   Set env vars? (yes/no): " SET_ENV

if [ "$SET_ENV" = "yes" ] || [ "$SET_ENV" = "y" ]; then
    echo ""
    echo "📝 Setting environment variables..."
    echo "   (You'll be prompted to enter each value)"
    echo ""

    # Load from .env if it exists
    if [ -f .env ]; then
        source .env
    fi

    # PIPEDRIVE_API_KEY
    if [ -n "$PIPEDRIVE_API_KEY" ]; then
        echo "PIPEDRIVE_API_KEY=$PIPEDRIVE_API_KEY" | vercel env add PIPEDRIVE_API_KEY production
    else
        vercel env add PIPEDRIVE_API_KEY production
    fi

    # PIPEDRIVE_BASE_URL
    echo "PIPEDRIVE_BASE_URL=https://api.pipedrive.com/v1" | vercel env add PIPEDRIVE_BASE_URL production

    # RETELL_API_KEY
    if [ -n "$RETELL_API_KEY" ]; then
        echo "RETELL_API_KEY=$RETELL_API_KEY" | vercel env add RETELL_API_KEY production
    else
        vercel env add RETELL_API_KEY production
    fi

    # RETELL_ASSISTANT_ID
    if [ -n "$RETELL_ASSISTANT_ID" ]; then
        echo "RETELL_ASSISTANT_ID=$RETELL_ASSISTANT_ID" | vercel env add RETELL_ASSISTANT_ID production
    else
        vercel env add RETELL_ASSISTANT_ID production
    fi

    # RETELL_FROM_NUMBER
    if [ -n "$RETELL_FROM_NUMBER" ]; then
        echo "RETELL_FROM_NUMBER=$RETELL_FROM_NUMBER" | vercel env add RETELL_FROM_NUMBER production
    else
        vercel env add RETELL_FROM_NUMBER production
    fi

    # RETELL_BASE_URL
    echo "RETELL_BASE_URL=https://api.retellai.com" | vercel env add RETELL_BASE_URL production

    # GIN_MODE
    echo "GIN_MODE=release" | vercel env add GIN_MODE production

    echo ""
    echo "✅ Environment variables set!"
    echo ""
fi

# Test locally first
echo "🧪 Testing local build..."
cd api
if go build -o /tmp/test-build index.go; then
    rm -f /tmp/test-build
    echo "✅ Local build successful!"
    cd ..
else
    echo "❌ Local build failed! Fix errors before deploying."
    cd ..
    exit 1
fi
echo ""

# Deploy to Vercel
echo "🚀 Deploying to Vercel (Production)..."
echo ""
vercel --prod

# Get deployment URL
echo ""
echo "✅ Deployment complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Test your deployment:"
echo "      curl https://your-url.vercel.app/health"
echo ""
echo "   2. Configure Pipedrive webhook:"
echo "      URL: https://your-url.vercel.app/webhook/pipedrive/lead"
echo "      Event: Lead - created"
echo ""
echo "   3. Configure Retell AI webhook:"
echo "      URL: https://your-url.vercel.app/webhook/retell/analyzed"
echo "      Events: call_analyzed, call_started, call_ended"
echo ""
echo "📖 See VERCEL_DEPLOYMENT.md for detailed instructions"
echo ""
