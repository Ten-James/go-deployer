# Go Deploy System

A simple client-server deployment system that allows you to deploy local projects to a remote server.

## Components

- **Server** (`server.go`): Receives deployment packages, extracts them, and runs deployment scripts
- **Client** (`client.go`): Packages local directory and sends it to the server for deployment

## Usage

### Building

```bash
# Build both server and client
make build

# Or build individually
make build-server
make build-client
```

### Running the Server

```bash
# Start the deployment server
make run-server
# or
./deploy-server
```

The server will listen on port 8080 by default.

### Deploying from Client

```bash
# Deploy current directory to server
./go-deploy http://your-server:8080
```

## Requirements

1. Your project directory must contain a `DEPLOY.sh` script
2. The `DEPLOY.sh` script will be executed on the server after extraction
3. The server automatically cleans up deployment files after execution

## Security Notes

- The server runs deployment scripts with bash
- Files are temporarily stored in `./uploads/` directory
- Automatic cleanup occurs 5 seconds after deployment completion
- Common files/directories are excluded from deployment package (.git, node_modules, etc.)

## Example DEPLOY.sh

```bash
#!/bin/bash
echo "Starting deployment..."

# Build Docker image
docker build -t my-app:latest .

# Stop and remove existing container
docker stop my-app 2>/dev/null || true
docker rm my-app 2>/dev/null || true

# Start new container
docker run -d --name my-app -p 3000:3000 my-app:latest

echo "Deployment completed!"
```