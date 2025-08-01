package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIKey    string `json:"api_key"`
	ServerURL string `json:"server_url"`
}

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "help" {
			fmt.Println("Usage: go-deploy <server-url|config> [options]")
			fmt.Println("Commands:")
			fmt.Println("  go-deploy config <api-key> <server-url>  - Set API key and server URL")
			fmt.Println("  go-deploy <server-url>                   - Deploy to specified server")
			fmt.Println("  go-deploy                                - Deploy using config server URL")
			os.Exit(1)
		}

		if os.Args[1] == "config" {
			if len(os.Args) < 4 {
				fmt.Println("Usage: go-deploy config <api-key> <server-url>")
				os.Exit(1)
			}
			if err := saveConfig(os.Args[2], os.Args[3]); err != nil {
				fmt.Printf("Failed to save config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Configuration saved successfully")
			return
		}
	}

	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		fmt.Println("Run 'go-deploy config <api-key> <server-url>' to set your configuration")
		os.Exit(1)
	}

	var serverURL string
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	} else {
		if config.ServerURL == "" {
			fmt.Println("No server URL provided. Use 'go-deploy <server-url>' or set it in config")
			os.Exit(1)
		}
		serverURL = config.ServerURL
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	deployScript := filepath.Join(currentDir, "DEPLOY.sh")
	if _, err := os.Stat(deployScript); os.IsNotExist(err) {
		fmt.Println("DEPLOY.sh not found in current directory")
		os.Exit(1)
	}

	fmt.Println("Creating deployment package...")
	zipData, err := createZip(currentDir)
	if err != nil {
		fmt.Printf("Failed to create deployment package: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sending deployment to %s...\n", serverURL)
	if err := sendDeployment(serverURL, zipData, config.APIKey); err != nil {
		fmt.Printf("Failed to send deployment: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Deployment sent successfully!")
}

func createZip(sourceDir string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if shouldSkip(path, sourceDir) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFile, file)
		return err
	})

	if err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func shouldSkip(path, sourceDir string) bool {
	relPath, _ := filepath.Rel(sourceDir, path)
	
	skipPaths := []string{
		".git",
		"node_modules",
		".env",
		"*.log",
		".DS_Store",
		"Thumbs.db",
	}

	for _, skip := range skipPaths {
		if strings.Contains(relPath, skip) {
			return true
		}
	}

	return false
}

func sendDeployment(serverURL string, zipData []byte, apiKey string) error {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("deployment", "deployment.zip")
	if err != nil {
		return err
	}

	if _, err := part.Write(zipData); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	deployURL := serverURL + "/deploy"
	req, err := http.NewRequest("POST", deployURL, &requestBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Print(string(body))
	return nil
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, ".go-deploy")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

func saveConfig(apiKey, serverURL string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	config := Config{
		APIKey:    apiKey,
		ServerURL: serverURL,
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func loadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API key not found in config")
	}

	return &config, nil
}