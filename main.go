// This program watches a directory or file for file changes and sends HTTP requests based on a template file, with file extention .http
package main

import (
	"log"
)

type HTTPFile struct {
	Method      string
	URL         string
	Headers     map[string]string // "hello":"world"
	Body        string
	ContentType string
}

func main() {
	config, err := flagsConfig()
	if err != nil {
		log.Fatalf("Error parsing configuration: %v", err)
	}
	logVerbose(config, "Configuration loaded successfully")

	processHTTPFiles(config)

	// // Load HTTP templates
	// var httpFileContents []HTTPFileContent
	// httpFileContents, err = readHTTPFile(config)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// Additional code for watching files and sending HTTP requests would go here
}
