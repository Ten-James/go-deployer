#!/bin/bash

go build -o server server.go
if [ $? -ne 0 ]; then
    echo "Build failed"
    exit 1
fi
echo "Build successful"
sudo mv server /usr/local/bin/go-deployer
if [ $? -ne 0 ]; then
    echo "Failed to move binary to /usr/local/bin"
    exit 1
fi
echo "Binary moved to /usr/local/bin/go-deployer"
