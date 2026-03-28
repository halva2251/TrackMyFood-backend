#!/bin/bash
set -e

REPO="https://github.com/halva2251/TrackMyFood-backend.git"
DIR="TrackMyFood-backend"

# Clone if not already present
if [ ! -d "$DIR" ]; then
  echo "Cloning repository..."
  git clone "$REPO"
else
  echo "Repository already exists, pulling latest changes..."
  git -C "$DIR" pull
fi

cd "$DIR"

# Ensure .env exists (use example if missing)
if [ ! -f .env ]; then
  echo "Creating .env from .env.example..."
  cp .env.example .env
fi

# Stop old containers
echo "Cleaning up old containers..."
docker-compose down || true

# Build and start containers
# Using --build to force local compilation of your new AI code
echo "Building and starting containers..."
docker-compose up --build -d

echo ""
echo "API is available at http://localhost:8090"
echo "Chat Test: curl -X POST http://localhost:8090/api/scan/7640150491001/chat -H 'Content-Type: application/json' -d '{\"question\": \"Is this safe?\"}'"
