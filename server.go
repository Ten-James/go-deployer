package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	uploadDir = "./uploads"
	port      = ":9999"
)

var apiKey string

func main() {
	flag.StringVar(&apiKey, "api-key", "", "API key for authentication")
	flag.Parse()

	if apiKey == "" {
		log.Fatal("API key is required. Use -api-key flag")
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	http.HandleFunc("/deploy", authenticateMiddleware(handleDeploy))
	
	fmt.Printf("Deployment server listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("deployment")
	if err != nil {
		http.Error(w, "Failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	timestamp := time.Now().Format("20060102-150405")
	deployDir := filepath.Join(uploadDir, fmt.Sprintf("deploy-%s", timestamp))
	
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		http.Error(w, "Failed to create deployment directory", http.StatusInternalServerError)
		return
	}

	zipPath := filepath.Join(deployDir, "deployment.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		http.Error(w, "Failed to create zip file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		http.Error(w, "Failed to save zip file", http.StatusInternalServerError)
		return
	}

	extractDir := filepath.Join(deployDir, "extracted")
	if err := unzip(zipPath, extractDir); err != nil {
		http.Error(w, "Failed to extract deployment", http.StatusInternalServerError)
		return
	}

	deployScript := filepath.Join(extractDir, "DEPLOY.sh")
	if _, err := os.Stat(deployScript); os.IsNotExist(err) {
		http.Error(w, "DEPLOY.sh not found in deployment", http.StatusBadRequest)
		return
	}

	if err := os.Chmod(deployScript, 0755); err != nil {
		http.Error(w, "Failed to make deploy script executable", http.StatusInternalServerError)
		return
	}

	go func() {
		defer cleanup(deployDir)
		
		cmd := exec.Command("/bin/bash", "DEPLOY.sh")
		cmd.Dir = extractDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			log.Printf("Deploy script failed: %v", err)
		} else {
			log.Printf("Deploy script completed successfully")
		}
	}()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Deployment started successfully in %s\n", deployDir)
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			return nil
		}

		os.MkdirAll(filepath.Dir(path), 0755)

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, rc)
		return err
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func authenticateMiddleware(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != apiKey {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func cleanup(dir string) {
	time.Sleep(5 * time.Second)
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("Failed to cleanup directory %s: %v", dir, err)
	} else {
		log.Printf("Cleaned up deployment directory: %s", dir)
	}
}