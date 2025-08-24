#!/bin/bash

while true; do
    echo "Pulling latest changes..."
    git pull

    echo "Tidying Go modules..."
    go mod tidy

    echo "Building the Go project..."
    go build -o discord-bot

    if [ $? -eq 0 ]; then
        echo "Build succeeded. Running the executable..."
        ./discord-bot
        echo "Executable exited with code $?. Restarting..."
    else
        echo "Build failed. Retrying in 5 seconds..."
        sleep 5
    fi
done