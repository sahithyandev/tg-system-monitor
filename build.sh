#!/bin/bash

# Build script for TG System Monitor
# This script generates default configuration from YAML and builds the optimized binary

set -e

echo "🔧 Generating default configuration from default-config.yml..."
go run tools/generate_defaults.go

echo "🏗️  Building optimized binary..."
go build -trimpath -buildmode=pie -gcflags="-l -B -C" -ldflags="-s -w -extldflags=-static" -o tgsm .

echo "📦 Binary size:"
ls -lh tgsm

echo "✅ Build complete!"
