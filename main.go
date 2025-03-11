// This program watches a directory or file for file changes and sends HTTP requests based on a template file, with file extention .http
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type HTTPFile struct {
	Method      string
	URL         string
	Headers     map[string]string // "hello":"world"
	Body        string
	ContentType string
}

func main() {
	config, err := parseConfig()
	if err != nil {
		log.Fatalf("Error parsing configuration: %v", err)
	}
	logVerbose(config, "Configuration loaded successfully")

	// Load HTTP templates
	httpFiles, err := readHTTPFile(config)
	if err != nil {
		fmt.Println(err)
	}

	// Print the content and path of each HTTP file
	for i, file := range httpFiles {
		fmt.Println("Content:", i, file.Content)
		fmt.Println("Path:", i, file.FilePath)
	}

	// Additional code for watching files and sending HTTP requests would go here
}

// parseHTTPFile reads and parses an HTTP template file
// receives string and returns a string of HTTP
// func parseHTTPFile(filepath string) (*HTTPFile, error) {
//
// }

// HTTPFileContent represents the content of an HTTP file along with its path
type HTTPFileContent struct {
	Content  string
	FilePath string
}

func readHTTPFile(config *Config) ([]HTTPFileContent, error) {
	var filePaths []string
	var results []HTTPFileContent
	logVerbose(config, "Reading http files...")

	// Check if a specific HTTP file is provided
	if config.HTTPFilePath != "" {
		// Convert to absolute path if needed
		absPath, err := filepath.Abs(config.HTTPFilePath)
		if err != nil {
			return nil, fmt.Errorf("error getting absolute path for %s: %v", config.HTTPFilePath, err)
		}
		filePaths = append(filePaths, absPath)
	} else if config.HTTPFolderPath != "" {
		// Convert folder path to absolute path
		absFolderPath, err := filepath.Abs(config.HTTPFolderPath)
		if err != nil {
			return nil, fmt.Errorf("error getting absolute path for %s: %v", config.HTTPFolderPath, err)
		}

		// Read all .http files from the specified folder
		files, err := os.ReadDir(absFolderPath)
		if err != nil {
			return nil, fmt.Errorf("error reading directory %s: %v", absFolderPath, err)
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".http") {
				filePaths = append(filePaths, filepath.Join(absFolderPath, file.Name()))
			}
		}

		if len(filePaths) == 0 {
			return nil, fmt.Errorf("no .http files found in directory %s", absFolderPath)
		}
	} else {
		// Find .http file in current directory if no file or folder specified
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting current directory: %v", err)
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("error reading directory: %v", err)
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".http") {
				filePaths = append(filePaths, filepath.Join(dir, file.Name()))
				break // Only use the first .http file found when in current directory
			}
		}

		if len(filePaths) == 0 {
			return nil, fmt.Errorf("no .http file found in the current directory")
		}
	}

	// Process each file
	for _, filePath := range filePaths {
		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading file %s: %v", filePath, err)
		}
		fileContent := string(content)

		// Add this file to the results
		results = append(results, HTTPFileContent{
			Content:  fileContent,
			FilePath: filePath,
		})
	}

	return results, nil
}
