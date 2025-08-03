package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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

		zipPath := strings.ReplaceAll(relPath, string(filepath.Separator), "/")
		zipFile, err := zipWriter.Create(zipPath)
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