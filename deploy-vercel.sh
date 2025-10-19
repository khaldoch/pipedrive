#!/bin/bash

echo "🚀 Deploying PipCal to Vercel..."

# Check if Vercel CLI is installed
if ! command -v vercel &> /dev/null; then
    echo "❌ Vercel CLI not found. Installing..."
    npm install -g vercel
fi

# Copy all necessary files to api directory
echo "📁 Copying files to api directory..."
cp *.go api/
cp go.mod go.sum api/

# Deploy to Vercel
echo "🌐 Deploying to Vercel..."
vercel --prod

echo "✅ Deployment complete!"
echo "🔗 Your webhooks are now available at:"
echo "   - https://your-app.vercel.app/webhook/retell"
echo "   - https://your-app.vercel.app/webhook/cal"
echo "   - https://your-app.vercel.app/webhook/retell/analyzed"
echo "   - https://your-app.vercel.app/webhook/pipedrive/lead"
echo ""
echo "🧪 Test endpoints:"
echo "   - https://your-app.vercel.app/test/completed"
echo "   - https://your-app.vercel.app/test/pipedrive-lead"
