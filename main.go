// This program watches a directory or file for file changes and sends HTTP requests based on a template file, with file extention .http
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	verboseLogger *log.Logger = log.New(os.Stdout, "Debug: ", log.LstdFlags)
	config        *Config
)

// Configuration holds the program settings
type Config struct {
	WatchFolderPath string // folder to be watch for changes
	WatchFilePath   string // File to be watched for changes
	// .http files must be watched for changes as well, and if are changed, the program must be update.
	HTTPFilePath    string // optional, if no file path is passed, it must search the first one on the same directory program was run.
	HTTPFolderPath  string // optional, if no folder path is passed, it must search all .http files in the same directory program was run.
	ExcludeFile     string // this can be an exact folder or a pattern of files, which means this files won't be watched.
	ExcludeFolder   string // this means any file inside this folder will be ignore or not watched.
	WaitRequestTime int    // Time to wait in between the HTTP requests
	MaxConcurrent   int    // one at the same time, we want the requests to go one after the other, one must fish before other one is sent.
	Verbose         bool   // Detailed Loging for debugging purposes
}

func logVerbose(format string, args ...any) {
	if config.Verbose {
		verboseLogger.Printf(format, args...)
	}
}
func parseConfig() (*Config, error) {
	// Default configuration values
	config = &Config{
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

	flag.Visit(func(f *flag.Flag) {
		logVerbose("Flag passed: %s = %s", f.Name, f.Value.String())
	})

	// Validate configuration
	if config.WatchFolderPath == "" && config.WatchFilePath == "" {
		return nil, fmt.Errorf("either watch-folder or watch-file must be specified")
	}

	if config.HTTPFilePath != "" {
		info, err := checkPathExists(config.HTTPFilePath)
		if err != nil {
			return nil, fmt.Errorf("file path does not exists")
		}
		if info.IsDir() {
			return nil, fmt.Errorf("provided http file but is directory: %s", config.HTTPFilePath)
		}
	}

	if config.HTTPFolderPath != "" {
		info, err := checkPathExists(config.HTTPFolderPath)
		if err != nil {
			return nil, fmt.Errorf("folder path does not exists")
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("provided http folder is not a directory: %s", config.HTTPFolderPath)

		}
	}

	if config.WaitRequestTime < 0 {
		return nil, fmt.Errorf("wait-time cannot be negative")
	}

	if config.MaxConcurrent < 1 {
		return nil, fmt.Errorf("max-concurrent must be at least 1")
	}

	return config, nil
}
func checkPathExists(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("http file doesnot exist: %s", path)
		}
		return nil, fmt.Errorf("error checking http file: %w", err)
	}
	return info, nil
}

func main() {
	fmt.Println("hello")
	_, err := parseConfig()
	if err != nil {
		log.Fatalf("Error parsing configuration: %v", err)
	}
	fmt.Println("hello")

	// logVerbose("Configuration loaded successfully")

	// Log only the flags that were explicitly passed
	//logPassedFlags()

}
