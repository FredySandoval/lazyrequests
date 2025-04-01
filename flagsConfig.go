package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Configuration holds the program settings
type Config struct {
	WatchFolderPath string // folder to be watch for changes
	WatchFilePath   string // File to be watched for changes
	// .http files must be watched for changes as well, and if are changed, the program must be update.
	HTTPFilePath       string // optional, if no file path is passed, it must search the first one on the same directory program was run.
	HTTPFolderPath     string // optional, if no folder path is passed, it must search all .http files in the same directory program was run.
	ExcludeFile        string // this can be an exact folder or a pattern of files, which means this files won't be watched.
	ExcludeFolder      string // this means any file inside this folder will be ignore or not watched.
	WaitRequestTime    int    // Time to wait in between the HTTP requests
	HTTPRequestTimeOut int    // Time each request waits before considered failed
	Verbose            bool   // Detailed Loging for debugging purposes
}

func logVerbose(config *Config, format string, args ...any) {
	if config.Verbose {
		verboseLogger.Printf(format, args...)
	}
}

// isValidHttpExtension checks if the file has a valid HTTP request extension (.http or .rest)
func isValidHttpExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".http" || ext == ".rest"
}

func flagsConfig() (*Config, error) {
	// Default configuration values
	config := &Config{
		WatchFolderPath:    "",
		WatchFilePath:      "",
		HTTPFilePath:       "",
		HTTPFolderPath:     "",
		ExcludeFile:        "",
		ExcludeFolder:      "",
		WaitRequestTime:    50,   // Time in between requsts Default 50 milliseconds for developement
		HTTPRequestTimeOut: 3000, // default 3 seconds
		Verbose:            false,
	}

	// Parse command line flags
	flag.StringVar(&config.WatchFolderPath, "watch-folder", config.WatchFolderPath, "Folder to watch for changes")
	flag.StringVar(&config.WatchFilePath, "watch-file", config.WatchFilePath, "File to watch for changes")
	flag.StringVar(&config.HTTPFilePath, "http-file", config.HTTPFilePath, "HTTP template file to be watched and reloaded")
	flag.StringVar(&config.HTTPFolderPath, "http-folder", config.HTTPFolderPath, "HTTP template folder to be watched and reloaded")
	flag.StringVar(&config.ExcludeFile, "exclude-file", config.ExcludeFile, "File pattern to exclude from watching")
	flag.StringVar(&config.ExcludeFolder, "exclude-folder", config.ExcludeFolder, "Folder to exclude from watching")
	flag.IntVar(&config.WaitRequestTime, "wait-time", config.WaitRequestTime, "Time to wait between each HTTP requests (milliseconds)")
	flag.IntVar(&config.HTTPRequestTimeOut, "req-time-out", config.WaitRequestTime, "Timeout for each request before failing (milliseconds)")
	flag.BoolVar(&config.Verbose, "verbose", config.Verbose, "Enable verbose logging")

	flag.Parse()

	if config.Verbose {
		flag.Visit(func(f *flag.Flag) {
			logVerbose(config, "Flag passed: %s = %s", f.Name, f.Value.String())
		})
	}

	// Validate configuration
	if config.WatchFolderPath == "" && config.WatchFilePath == "" {
		return nil, fmt.Errorf("either --watch-folder or --watch-file must be specified")
	}
	fmt.Println(config.ExcludeFile)
	// Check for logical dependencies between arguments
	if config.ExcludeFile != "" && config.WatchFolderPath == "" {
		return nil, fmt.Errorf("exclude-file only makes sense when --watch-folder is specified")
	}

	// Validate watch paths
	if config.WatchFolderPath != "" {
		info, err := checkPathExists(config.WatchFolderPath)
		if err != nil {
			return nil, fmt.Errorf("watch folder error: %w", err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("provided watch folder is not a directory: %s", config.WatchFolderPath)
		}
	}

	if config.WatchFilePath != "" {
		info, err := checkPathExists(config.WatchFilePath)
		if err != nil {
			return nil, fmt.Errorf("watch file error: %w", err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("provided watch file is a directory: %s", config.WatchFilePath)
		}

		// Validate file extension for watch file
		if !isValidHttpExtension(config.WatchFilePath) {
			return nil, fmt.Errorf("watch file must have .http or .rest extension: %s", config.WatchFilePath)
		}
	}

	// Validate HTTP paths
	if config.HTTPFilePath != "" {
		info, err := checkPathExists(config.HTTPFilePath)
		if err != nil {
			return nil, fmt.Errorf("http file error: %w", err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("provided http file is a directory: %s", config.HTTPFilePath)
		}

		// Validate file extension for HTTP file
		if !isValidHttpExtension(config.HTTPFilePath) {
			return nil, fmt.Errorf("HTTP file must have .http or .rest extension: %s", config.HTTPFilePath)
		}
	}

	if config.HTTPFolderPath != "" {
		info, err := checkPathExists(config.HTTPFolderPath)
		if err != nil {
			return nil, fmt.Errorf("http folder error: %w", err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("provided http folder is not a directory: %s", config.HTTPFolderPath)
		}
	}

	if config.HTTPFilePath == "" && config.HTTPFolderPath == "" {
		logVerbose(config, "no .http file or files provided...")
	}

	if config.WaitRequestTime < 0 {
		return nil, fmt.Errorf("wait-time cannot be negative")
	}

	return config, nil
}

func checkPathExists(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		return nil, fmt.Errorf("error checking path: %w", err)
	}
	return info, nil
}
