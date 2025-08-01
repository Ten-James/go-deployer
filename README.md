# 🚀 Go Deploy System

A secure, cross-platform client-server deployment system for automating project deployments to remote Linux servers.

## ✨ Features

- 🔒 **Secure Authentication** - API key-based authentication
- 📦 **Smart Packaging** - Automatic zip creation with intelligent file exclusion
- 🧹 **Auto Cleanup** - Temporary files cleaned up automatically
- 🌐 **Cross-Platform Client** - Works on Windows, macOS, and Linux
- ⚡ **Fast Deployment** - Efficient file transfer and extraction
- 🛡️ **Path Security** - Protected against zip slip attacks

## 📁 Project Structure

```
go-web-ssh/
├── server.go          # Deployment server (Linux)
├── client.go          # Cross-platform client
├── go.mod             # Go module definition
├── Makefile           # Build automation
├── DEPLOY.sh          # Example deployment script
├── index.html         # Server status page
└── README.md          # This file
```

## 🚀 Quick Start

### 1. Build the Applications

```bash
# Build both server and client
make build

# Or build individually
make build-server
make build-client
```

### 2. Configure the Client

```bash
# Set your API key and server URL
./go-deploy config <your-api-key> http://your-server:9999
```

### 3. Start the Server (Linux)

```bash
# Start with your API key
./deploy-server -api-key=<your-api-key>
```

### 4. Deploy Your Project

```bash
# Deploy current directory
./go-deploy

# Or deploy to specific server
./go-deploy http://your-server:9999
```

## 📋 Requirements

### Server Requirements
- ✅ Linux operating system
- ✅ Bash shell available
- ✅ Network connectivity on port 9999

### Client Requirements
- ✅ Windows, macOS, or Linux
- ✅ Go 1.21+ (for building)
- ✅ Project must contain `DEPLOY.sh` script

### Project Requirements
- 📝 `DEPLOY.sh` script in project root
- 🔧 Script must be executable on Linux
- 📂 Valid project structure

## 🔧 Configuration

The client stores configuration in `~/.go-deploy/config.json`:

```json
{
  "api_key": "your-secret-key",
  "server_url": "http://your-server:9999"
}
```

## 🛡️ Security Features

- 🔐 **API Key Authentication** - Bearer token authorization
- 🚫 **File Exclusion** - Automatically excludes sensitive files:
  - `.git/` - Git repository data
  - `node_modules/` - Node.js dependencies
  - `.env` - Environment variables
  - `*.log` - Log files
  - `.DS_Store`, `Thumbs.db` - OS metadata
- 🛡️ **Path Validation** - Protection against directory traversal attacks
- ⏰ **Automatic Cleanup** - Temporary files removed after 5 seconds
- 🔒 **Secure Permissions** - Proper file permissions (0755, 0600)

## 📝 Example DEPLOY.sh

```bash
#!/bin/bash
set -e  # Exit on any error

echo "🚀 Starting deployment..."

# Build Docker image
echo "📦 Building Docker image..."
docker build -t my-app:latest .

# Stop existing container
echo "🛑 Stopping existing container..."
docker stop my-app 2>/dev/null || true
docker rm my-app 2>/dev/null || true

# Start new container
echo "▶️  Starting new container..."
docker run -d --name my-app -p 3000:3000 my-app:latest

# Health check
echo "🏥 Performing health check..."
sleep 5
if curl -f http://localhost:3000/health; then
    echo "✅ Deployment completed successfully!"
else
    echo "❌ Health check failed!"
    exit 1
fi
```

## 🔍 API Reference

### Deploy Endpoint

**POST** `/deploy`

- **Headers**: `Authorization: Bearer <api-key>`
- **Content-Type**: `multipart/form-data`
- **Body**: Form field `deployment` with zip file
- **Response**: Deployment status and directory path

## 🐛 Troubleshooting

| Issue | Solution |
|-------|----------|
| **"API key is required"** | Use `-api-key` flag when starting server |
| **"DEPLOY.sh not found"** | Ensure script exists in project root |
| **"Permission denied"** | Check file permissions and API key |
| **"Connection refused"** | Verify server is running and port is open |

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test on multiple platforms
5. Submit a pull request

## 📄 License

This project is open source and available under the [MIT License](LICENSE).