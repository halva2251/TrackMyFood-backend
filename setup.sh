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

# Start containers
echo "Starting containers..."
docker compose up -d

echo ""
echo "API is available at http://localhost:8090"
echo "Try: curl http://localhost:8090/api/scan/7610000000001"
