#!/bin/bash

echo "Building the Go project..."

# Make sure modules are tidy
go mod tidy

# Build the executable
go build -o discord-bot

# Check if build succeeded
if [ $? -eq 0 ]; then
    echo "Build succeeded. Running the executable..."
    ./discord-bot
else
    echo "Build failed."
    exit 1
fi
