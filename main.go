// This program watches a directory or file for file changes and sends HTTP requests based on a template file, with file extention .http
package main

import (
	"flag"
	"fmt"
	"log"
)

var (
	config *Config
)

// Configuration holds the program settings
type Config struct {
	WatchFolderPath string // folder to be watch for changes
	WatchFilePath   string // File to be watched for changes
	HTTPFilePath    string // by default the file must be watched, which means if this file is changed, it must be reloaded.
	HTTPFolderPath  string // by default the folder must be watched, which means if any file in the folder is changed, it must be reloaded.
	ExcludeFile     string // this can be an exact folder or a pattern of files, which means this files won't be watched.
	ExcludeFolder   string // this means any file inside this folder will be ignore or not watched.
	WaitRequestTime int    // Time to wait in between the HTTP requests
	MaxConcurrent   int    // one at the same time, we want the requests to go one after the other, one must fish before other one is sent.
	Verbose         bool   // Detailed Loging for debugging purposes
}

func parseConfig() (*Config, error) {
	// Default configuration values
	config := &Config{
		WatchFolderPath: "",
		WatchFilePath:   "",
		HTTPFilePath:    "",
		HTTPFolderPath:  "",
		ExcludeFile:     "",
		ExcludeFolder:   "",
		WaitRequestTime: 1000, // Default 1 second in milliseconds
		MaxConcurrent:   1,    // Default to single concurrent request
		Verbose:         false,
	}

	// Parse command line flags
	flag.StringVar(&config.WatchFolderPath, "watch-folder", config.WatchFolderPath, "Folder to watch for changes")
	flag.StringVar(&config.WatchFilePath, "watch-file", config.WatchFilePath, "File to watch for changes")
	flag.StringVar(&config.HTTPFilePath, "http-file", config.HTTPFilePath, "HTTP template file to be watched and reloaded")
	flag.StringVar(&config.HTTPFolderPath, "http-folder", config.HTTPFolderPath, "HTTP template folder to be watched and reloaded")
	flag.StringVar(&config.ExcludeFile, "exclude-file", config.ExcludeFile, "File pattern to exclude from watching")
	flag.StringVar(&config.ExcludeFolder, "exclude-folder", config.ExcludeFolder, "Folder to exclude from watching")
	flag.IntVar(&config.WaitRequestTime, "wait-time", config.WaitRequestTime, "Time to wait between HTTP requests (milliseconds)")
	flag.IntVar(&config.MaxConcurrent, "max-concurrent", config.MaxConcurrent, "Maximum number of concurrent requests")
	flag.BoolVar(&config.Verbose, "verbose", config.Verbose, "Enable verbose logging")

	flag.Parse()

	// Validate configuration
	if config.WatchFolderPath == "" && config.WatchFilePath == "" {
		return nil, fmt.Errorf("either watch-folder or watch-file must be specified")
	}

	if config.HTTPFilePath == "" && config.HTTPFolderPath == "" {
		return nil, fmt.Errorf("either http-file or http-folder must be specified")
	}

	if config.WaitRequestTime < 0 {
		return nil, fmt.Errorf("wait-time cannot be negative")
	}

	if config.MaxConcurrent < 1 {
		return nil, fmt.Errorf("max-concurrent must be at least 1")
	}

	return config, nil
}
func logVerbose(format string, args ...interface{}) {
	if config.Verbose {
		log.Printf(format, args...)
	}
}

// logPassedFlags logs flags that were explicitly passed by the user
func logPassedFlags() {
	flag.Visit(func(f *flag.Flag) {
		logVerbose("Flag passed: %s = %s", f.Name, f.Value.String())

		// Additional verbose messages for specific flags
		switch f.Name {
		case "watch-folder":
			logVerbose("Watching folder for changes: %s", config.WatchFolderPath)
		case "watch-file":
			logVerbose("Watching file for changes: %s", config.WatchFilePath)
		case "http-file":
			logVerbose("Using HTTP template file: %s", config.HTTPFilePath)
		case "http-folder":
			logVerbose("Using HTTP template folder: %s", config.HTTPFolderPath)
		case "exclude-file":
			logVerbose("Excluded file pattern: %s", config.ExcludeFile)
		case "exclude-folder":
			logVerbose("Excluded folder: %s", config.ExcludeFolder)
		case "wait-time":
			logVerbose("Wait time between requests: %d ms", config.WaitRequestTime)
		case "max-concurrent":
			logVerbose("Maximum concurrent requests: %d", config.MaxConcurrent)
		}
	})
}
func main() {
	var err error
	config, err = parseConfig()
	if err != nil {
		log.Fatalf("Error parsing configuration: %v", err)
	}

	logVerbose("Configuration loaded successfully")

	// Log only the flags that were explicitly passed
	logPassedFlags()

}
