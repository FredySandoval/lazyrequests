// This program watches a directory or file for file changes and sends HTTP requests based on a template file, with file extention .http
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HTTPFile struct {
	Method      string
	URL         string
	Headers     map[string]string // "hello":"world"
	Body        string
	ContentType string
}

var (
	verboseLogger *log.Logger = log.New(os.Stdout, "Debug: ", log.LstdFlags)
)

const (
	C_Red       = "\033[31m"
	C_Green     = "\033[32m"
	C_Blue      = "\033[34m"
	C_Reset     = "\033[0m"
	C_White     = "\033[37m"
	C_Yellow    = "\033[33m"
	C_Purple    = "\033[35m"
	C_Cyan      = "\033[36m"
	C_Gray      = "\033[37m"
	C_Bold      = "\033[1m"
	C_Underline = "\033[4m"
)

// [x] TODO - tests might not be working,
// [x] TODO - only accept resclient valid formats in --httpfile
// [x] implement request response conparison
// [ ] implement the watch file function
func main() {
	config, err := flagsConfig()
	if err != nil {
		log.Fatalf("Error parsing configuration: %v", err)
	}
	logVerbose(config, "Configuration loaded successfully")

	httpFileContentParsed, err := processHTTPFiles(config)
	if err != nil {
		fmt.Println(err)
	}

	httpRequestTimeOut := time.Duration(config.HTTPRequestTimeOut) * time.Millisecond
	waitRequestTime := config.WaitRequestTime * int(time.Millisecond)
	for j, fileContent := range httpFileContentParsed {
		for k, block := range fileContent.Blocks {
			// Add a sleep to wait between requests
			if j > 0 || k > 0 {
				time.Sleep(time.Duration(waitRequestTime)) // Fixed 100ms wait between requests
			}

			reqDetails := block.Request
			ctx, cancel := context.WithTimeout(context.Background(), httpRequestTimeOut)

			newReq, err := http.NewRequestWithContext(ctx, reqDetails.Method, reqDetails.Url, strings.NewReader(reqDetails.Body))
			if err != nil {
				cancel()
				fmt.Printf("error at creating request: httpFileContentParsed[%d][%d]: %v", j, k, err)
			}
			// Add headers
			for key, value := range reqDetails.Headers {
				newReq.Header.Set(key, value)
			}
			client := &http.Client{}
			resp, err := client.Do(newReq)
			if err != nil {
				fileName := filepath.Base(fileContent.FilePath)
				fmt.Printf("%s: %s%v%s\n", fileName, C_Red, err, C_Reset)
				cancel()
				continue
			}
			results := struct {
				OK       bool
				MSG      string
				Expected string
				Got      string
			}{
				OK: true,
			}

			if block.ExpectedResponse != nil { // if response
				if block.ExpectedResponse.Status != "" { // if expected response status
					// if response status are the same
					if block.ExpectedResponse.Status != resp.Status {
						results.OK = false
						results.MSG = "Response Status mismatch"
						results.Expected = block.ExpectedResponse.Status
						results.Got = resp.Status
					}
				}
			}
			if results.OK {
				fmt.Printf("%s[✓][ %s ] %s %s %s  \n", C_Green, resp.Status, C_White, block.CommentIdentifier, C_Reset)
			} else {
				fmt.Printf("%s[X][ %s ] Expected: [ %s ] Got: [ %s ] %s \n", C_Red, results.MSG, results.Expected, results.Got, C_Reset)
			}
			// LOG IF WINS
			resp.Body.Close()
			cancel()
		}
	}
}
