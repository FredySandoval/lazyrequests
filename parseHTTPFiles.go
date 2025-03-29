package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type HTTPFileContent struct {
	RawContent      string
	FilePath        string
	GlobalVariables map[string]string
	Blocks          []HTTPBlock
}

type HTTPBlock struct {
	ID                     int
	BlockContent           string // represents the raw string request
	CommentIdentifier      string
	Request                *http.Request // represents the parsed request ready to be sent
	RequestString          string
	ExpectedResponse       *http.Response // represents the expected response to be compared with
	ExpectedResponseString string
}

type HTTPResponse struct {
	Protocol        string
	StatusCode      int
	StatusText      string
	ResponseHeaders map[string]string
	ResponseBody    string // not sure of type, so set as string but can be change later
}

func processHTTPFiles(config *Config) ([]HTTPFileContent, error) {
	httpFileContent, err := getRawContent(config)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - getRawContent() %w", err)
	}
	httpFileContent, err = removeComments(httpFileContent)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - removeComments() %w", err)
	}
	httpFileContent, err = getGlobalVariables(httpFileContent)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - getGlobalVariables() %w", err)
	}
	httpFileContent, err = parseBlocks(httpFileContent)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - parseBlocks() %w", err)
	}
	httpFileContent, err = checkNormalizationOnBlocks(httpFileContent, config)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - checkNormalizationOnBlocks() %w", err)
	}
	httpFileContent, err = handleHTTPRequestMultiline(httpFileContent, config)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - handleHTTPRequestMultiline() %w", err)
	}
	httpFileContent, err = validateHTTPRequestLine(httpFileContent, config)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - validateHTTPRequestLine() %w", err)
	}
	httpFileContent, err = parseHTTPBlockRequests(httpFileContent, config)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - parseHTTPBlockRequests() %w", err)
	}
	httpFileContent, err = parseHTTPBlockResponses(httpFileContent, config)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - parseHTTPBlockResponse() %w", err)
	}
	return httpFileContent, nil
}

// Helper function to check if a line is an HTTP response line
func isHTTPResponseLine(line string) bool {
	// HTTP response must start with HTTP protocol version followed by status code
	// Pattern: HTTP/X.X XXX ...
	pattern := regexp.MustCompile(`^HTTP/\d\.\d\s+\d{3}`)
	return pattern.MatchString(line)
}
func checkNormalizationOnBlocks(httpFileContent []HTTPFileContent, config *Config) ([]HTTPFileContent, error) {
	for i, file := range httpFileContent {
		for j, block := range file.Blocks {
			if !strings.HasSuffix(block.BlockContent, "\r\n\r\n") {
				trimmedContent := strings.TrimRight(block.BlockContent, "\r\n")
				normalizedContent := trimmedContent + "\r\n\r\n"
				httpFileContent[i].Blocks[j].BlockContent = normalizedContent
				logVerbose(config, fmt.Sprintf("Normalized block %d in file %s", block.ID, file.FilePath))
			}
		}
	}

	return httpFileContent, nil
}
func handleHTTPRequestMultiline(httpFileContent []HTTPFileContent, config *Config) ([]HTTPFileContent, error) {
	for i, file := range httpFileContent {
		for j, block := range file.Blocks {
			// Skip empty blocks
			if strings.TrimSpace(block.BlockContent) == "" {
				continue
			}

			// Split block content into lines
			lines := strings.Split(block.BlockContent, "\n")
			if len(lines) <= 1 {
				continue // Skip blocks with just one line
			}

			// Check if first line is an HTTP request using the helper function
			firstLine := strings.TrimSpace(lines[0])
			if !isHTTPRequestLine(firstLine) {
				continue
			}

			// Check if there are indented query parameter lines (starting with whitespace and ? or &)
			hasMultilineURL := false
			for k := 1; k < len(lines); k++ {
				trimmedLine := strings.TrimSpace(lines[k])
				if (strings.HasPrefix(trimmedLine, "?") || strings.HasPrefix(trimmedLine, "&")) &&
					(len(lines[k]) > len(trimmedLine)) { // Check if line starts with whitespace
					hasMultilineURL = true
					break
				}
			}

			if !hasMultilineURL {
				continue
			}

			// Process multi-line URL
			var newFirstLine string
			var remainingLines []string

			// Determine where the URL ends in the first line
			urlEnd := -1
			if strings.Contains(firstLine, " HTTP/") {
				urlEnd = strings.LastIndex(firstLine, " HTTP/")
			} else {
				urlEnd = len(firstLine)
			}

			baseURL := firstLine[:urlEnd]
			httpVersion := ""
			if urlEnd < len(firstLine) {
				httpVersion = firstLine[urlEnd:]
			}

			// Find all query parameter lines
			var queryParams []string
			k := 1
			for ; k < len(lines); k++ {
				trimmedLine := strings.TrimSpace(lines[k])
				if strings.HasPrefix(trimmedLine, "?") || strings.HasPrefix(trimmedLine, "&") {
					queryParams = append(queryParams, trimmedLine)
				} else {
					// Stop when we hit a non-query parameter line
					break
				}
			}

			// Collect remaining lines that aren't part of the URL
			remainingLines = lines[k:]

			// Build the new first line by combining the URL and query parameters
			if len(queryParams) > 0 {
				// Join all query parameters, ensuring ? appears only once at the start
				combinedQuery := strings.Join(queryParams, "")
				// If the first parameter already has a ? prefix and there's another ? in the query, replace it with &
				if strings.HasPrefix(combinedQuery, "?") && strings.Count(combinedQuery, "?") > 1 {
					combinedQuery = "?" + strings.ReplaceAll(combinedQuery[1:], "?", "&")
				}

				// Combine with the base URL
				newFirstLine = baseURL + combinedQuery + httpVersion
			} else {
				newFirstLine = firstLine
			}

			// Reconstruct block content with the new first line and remaining lines
			newBlockContent := newFirstLine
			if len(remainingLines) > 0 {
				newBlockContent += "\n" + strings.Join(remainingLines, "\n")
			}

			// Update the block content
			httpFileContent[i].Blocks[j].BlockContent = newBlockContent
		}
	}

	return httpFileContent, nil
}

// Helper function to check if a line is an HTTP request line
func isHTTPRequestLine(line string) bool {
	httpMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT"}
	trimmedLine := strings.TrimSpace(line)

	for _, method := range httpMethods {
		if strings.HasPrefix(trimmedLine, method+" ") {
			return true
		}
	}

	return false
}
func validateHTTPRequestLine(httpFileContent []HTTPFileContent, config *Config) ([]HTTPFileContent, error) {
	for i, file := range httpFileContent {
		for j, block := range file.Blocks {
			// Skip empty blocks
			if strings.TrimSpace(block.BlockContent) == "" {
				continue
			}

			// Split block content into lines
			lines := strings.Split(block.BlockContent, "\n")
			if len(lines) == 0 {
				continue
			}

			// Get the first line and check if it's an HTTP request using isHTTPRequestLine
			firstLine := strings.TrimSpace(lines[0])
			firstLineParts := strings.Fields(firstLine)

			method_line := firstLineParts[0]
			httpMethods_LowerCase := []string{"get", "post", "put", "delete", "patch", "head", "options"}
			isLowerCase := false
			// IF no method assumge GET
			for _, validMethod := range httpMethods_LowerCase {
				if method_line == validMethod {
					isLowerCase = true
					break
				}
			}
			if isLowerCase {
				firstLineParts[0] = strings.ToUpper(method_line)
				firstLine = strings.Join(firstLineParts, " ")
				lines[0] = firstLine
				httpFileContent[i].Blocks[j].BlockContent = strings.Join(lines, "\n")
			}

			method := strings.ToUpper(firstLineParts[0])
			httpMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
			isValidMethod := false
			// IF no method assumge GET
			for _, validMethod := range httpMethods {
				if method == validMethod {
					isValidMethod = true
					break
				}
			}
			if !isValidMethod {
				firstLine = "GET " + firstLine
			}

			// Check if HTTP version is missing (should have "HTTP/X.X" at the end)
			if !strings.Contains(firstLine, " HTTP/") {
				// Add HTTP/1.1 to the first line
				lines[0] = firstLine + " HTTP/1.1"

				// Reconstruct the block content with the fixed first line
				httpFileContent[i].Blocks[j].BlockContent = strings.Join(lines, "\n")
				logVerbose(config, fmt.Sprintf("validated, %s", lines))
			}

			// Skip if not a request
			if !isHTTPRequestLine(firstLine) {
				fmt.Println(j, "empty 1")
				continue
			}

		}
	}

	return httpFileContent, nil
}

func parseHTTPBlockRequests(httpFileContent []HTTPFileContent, config *Config) ([]HTTPFileContent, error) {
	for i, file := range httpFileContent {
		for j, block := range file.Blocks {
			// Skip empty blocks
			if strings.TrimSpace(block.BlockContent) == "" {
				continue
			}
			// Check if the block is a request by examining the first line
			lines := strings.Split(strings.TrimSpace(block.BlockContent), "\n")
			if len(lines) == 0 {
				continue
			}
			firstLine := lines[0]
			// Use isHTTPRequestLine to determine if this is a request
			if !isHTTPRequestLine(firstLine) {
				continue // Skip if not a request (likely a response)
			}

			// I do this to add the Content-Length properly
			ptr, err := stringToHTTPStruct(block.BlockContent)
			if err != nil {
				return nil, fmt.Errorf("error parsing HTTP request in file %s, block %d: %w", file.FilePath, block.ID, err)
			}

			// Create a new http request with proper timeout
			ctx := context.Background()
			timeout := 5 * time.Second

			if config.HTTPRequestTimeOut > 0 {
				timeout = time.Duration(config.HTTPRequestTimeOut) * time.Second
			}
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Create the request with body
			newReq, err := http.NewRequestWithContext(ctx, ptr.Method, ptr.Url.String(), strings.NewReader(ptr.Body))
			if err != nil {
				return nil, fmt.Errorf("error creating HTTP request in file %s, block %d: %w", file.FilePath, block.ID, err)
			}

			// Add headers
			for key, value := range ptr.Headers {
				newReq.Header.Set(key, value)
			}

			// Generate the request string for logging/debugging
			var requestStr strings.Builder
			requestStr.WriteString(fmt.Sprintf("%s %s %s\r\n", ptr.Method, ptr.Url.String(), ptr.HTTPVersion))
			for key, value := range ptr.Headers {
				requestStr.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
			}
			requestStr.WriteString("\r\n")
			requestStr.WriteString(ptr.Body)

			// Add request to the block
			httpFileContent[i].Blocks[j].Request = newReq
			httpFileContent[i].Blocks[j].RequestString = requestStr.String()
		}
	}
	return httpFileContent, nil
}

func removeComments(httpFileContent []HTTPFileContent) ([]HTTPFileContent, error) {
	// List of exceptions where the comment should be preserved.
	exceptions := []string{
		"// @prompt",
		"// @name",
		"// @note",
		"// @no-redirect",
		"// @no-cookie-jar",
	}

	for i, file := range httpFileContent {
		lines := strings.Split(file.RawContent, "\n")
		var filteredLines []string

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				// Check if the line starts with any of the exceptions.
				exceptionFound := false
				for _, exception := range exceptions {
					if strings.HasPrefix(trimmed, exception) {
						exceptionFound = true
						break
					}
				}
				// Skip the line only if it's a comment and doesn't match an exception.
				if !exceptionFound {
					continue
				}
			}
			filteredLines = append(filteredLines, line)
		}

		// Join the filtered lines back into the file's RawContent.
		httpFileContent[i].RawContent = strings.Join(filteredLines, "\n")
	}

	return httpFileContent, nil
}

//	func removeComments(HTTPFileContent []HTTPFileContent) ([]HTTPFileContent, error) {
//		// remove all the comments in the form of '// this is a comment'
//		// a comment is recognised because its starts with two consecutives // at the beginning of the line
//		// on this particular instance, comments cannot be in any other form like after some text, like `@var = somethingk // Don't parse me`
//		// only comments with // must be remove, any other form of comment like /***/ or # must be disregard
//		return HTTPFileContent, nil
//	}
func getGlobalVariables(httpFileContent []HTTPFileContent) ([]HTTPFileContent, error) {
	// Regular expression to capture lines like: @variable = value
	reVar := regexp.MustCompile(`^\s*@(\w+)\s*=\s*(.+)$`)
	// Regular expression to capture placeholders like: {{variable}} or {{variable/}}
	rePlaceholder := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	// Process each file
	for i, file := range httpFileContent {
		lines := strings.Split(file.RawContent, "\n")
		globals := make(map[string]string)
		var filteredLines []string

		// Scan each line for global variable definitions.
		for _, line := range lines {
			if matches := reVar.FindStringSubmatch(line); matches != nil {
				// matches[1] is the key, matches[2] is the value
				key := strings.TrimSpace(matches[1])
				value := strings.TrimSpace(matches[2])
				globals[key] = value
			} else {
				// Keep the line if it is not a global variable definition.
				filteredLines = append(filteredLines, line)
			}
		}

		// Join the filtered lines back into the file content.
		processedContent := strings.Join(filteredLines, "\n")

		// Replace all placeholders with their corresponding global variable values.
		processedContent = rePlaceholder.ReplaceAllStringFunc(processedContent, func(match string) string {
			// Extract the inner variable name from {{...}} and trim spaces.
			inner := strings.TrimSpace(match[2 : len(match)-2])
			// Direct match
			if val, ok := globals[inner]; ok {
				return val
			}
			// If the placeholder ends with a slash, try trimming it and match again.
			if strings.HasSuffix(inner, "/") {
				trimmed := strings.TrimSuffix(inner, "/")
				if val, ok := globals[trimmed]; ok {
					return val
				}
			}
			// If not found, return the original placeholder unchanged.
			return match
		})

		// Update the HTTPFileContent with the processed content and global variables.
		httpFileContent[i].RawContent = processedContent
		httpFileContent[i].GlobalVariables = globals
	}

	return httpFileContent, nil
}

//	[]HTTPFileContent{
//			{Content: "GET {{baseURL/}}/path HTTP/1.1", FilePath: "/path/to/File1"},
//			{Content: "GET {{baseURL/}}/path HTTP/1.1", FilePath: "/path/to/File2"},
//		}
func getRawContent(config *Config) ([]HTTPFileContent, error) {
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
			RawContent: fileContent,
			FilePath:   filePath,
		})
	}

	return results, nil
}

//	type rawHTTPFileContent struct {
//	    Content  string
//	    FilePath string
//	}
//
// returns parsed Content into blocks separated by delimiter "###"
func parseBlocks(httpFileContent []HTTPFileContent) ([]HTTPFileContent, error) {
	// Process each file in the array
	for i, fileContent := range httpFileContent {
		// Initialize variables
		var blocks []HTTPBlock
		var currentBlock strings.Builder
		var currentComment string
		var blockID int = 1

		// Check if fileContent is valid
		if fileContent.RawContent == "" {
			return nil, fmt.Errorf("empty file content in %s", fileContent.FilePath)
		}

		lines := strings.Split(fileContent.RawContent, "\n")

		// Handle the special case where text might start with a delimiter
		if len(lines) > 0 && strings.HasPrefix(lines[0], "###") {
			currentComment = strings.TrimSpace(strings.TrimPrefix(lines[0], "###"))
			lines = lines[1:]
		}

		// Process each line
		for lineNum, line := range lines {
			// Check if the line starts with the delimiter
			if strings.HasPrefix(line, "###") {
				// We found a delimiter, store the current block if not empty
				blockContent := strings.TrimSpace(currentBlock.String())
				if blockContent != "" {
					blocks = append(blocks, HTTPBlock{
						ID:                blockID,
						BlockContent:      blockContent,
						CommentIdentifier: currentComment,
					})
					blockID++
				}
				// Reset for the next block
				currentBlock.Reset()
				// Extract the comment after the delimiter
				currentComment = strings.TrimSpace(strings.TrimPrefix(line, "###"))
			} else {
				// Add the line to the current block
				if currentBlock.Len() > 0 {
					if _, err := currentBlock.WriteString("\n"); err != nil {
						return nil, fmt.Errorf("error writing newline at line %d in %s: %w", lineNum+1, fileContent.FilePath, err)
					}
				}
				if _, err := currentBlock.WriteString(line); err != nil {
					return nil, fmt.Errorf("error writing line %d in %s: %w", lineNum+1, fileContent.FilePath, err)
				}
			}
		}

		// Don't forget to add the last block if it's not empty
		blockContent := strings.TrimSpace(currentBlock.String())
		if blockContent != "" {
			blocks = append(blocks, HTTPBlock{
				ID:                blockID,
				BlockContent:      blockContent,
				CommentIdentifier: currentComment,
			})
		}

		// Check if any blocks were found
		if len(blocks) == 0 {
			return nil, fmt.Errorf("no valid blocks found in content of %s", fileContent.FilePath)
		}

		// Update the Blocks field in the httpFileContent
		httpFileContent[i].Blocks = blocks
	}

	return httpFileContent, nil
}

type HTTPRequest struct {
	Method      string
	Url         *url.URL
	HTTPVersion string
	Headers     map[string]string
	Body        string
}

func stringToHTTPStruct(requestString string) (HTTPRequest, error) {
	// Initialize default values for the request object
	defaultURL, _ := url.Parse("/")
	requestObject := HTTPRequest{
		Method:      "GET",
		Url:         defaultURL,
		HTTPVersion: "HTTP/1.1",
		Headers:     make(map[string]string),
		Body:        "",
	}

	// Normalize line endings and split the request string into lines
	normalizedString := strings.ReplaceAll(requestString, "\r\n", "##LINEBREAK##")
	normalizedString = strings.ReplaceAll(normalizedString, "\n", "##LINEBREAK##")
	lines := strings.Split(normalizedString, "##LINEBREAK##")

	// Parse the request line (first line)
	if len(lines) > 0 && lines[0] != "" {
		requestLineParts := strings.Split(lines[0], " ")

		// Check if the first part looks like a method
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"}
		isValidMethod := false
		for _, method := range validMethods {
			if requestLineParts[0] == method {
				isValidMethod = true
				break
			}
		}

		if isValidMethod {
			requestObject.Method = requestLineParts[0]

			// If there's a second part, it's the URL path
			if len(requestLineParts) >= 2 {
				parsedURL, err := url.Parse(requestLineParts[1])
				if err != nil {
					return requestObject, err
				}
				requestObject.Url = parsedURL
			}

			// If there's a third part, it's the HTTP version
			if len(requestLineParts) >= 3 {
				requestObject.HTTPVersion = requestLineParts[2]
			}
		}
	}

	// Parse headers
	headerIndex := 1
	bodyStartIndex := -1

	// Find where headers end (blank line) and collect headers
	for headerIndex < len(lines) {
		headerLine := lines[headerIndex]

		// Empty line indicates end of headers
		if headerLine == "" {
			bodyStartIndex = headerIndex + 1
			break
		}

		colonIndex := strings.Index(headerLine, ":")

		if colonIndex != -1 {
			headerName := strings.TrimSpace(headerLine[:colonIndex])
			headerValue := strings.TrimSpace(headerLine[colonIndex+1:])

			// Store header in object
			requestObject.Headers[headerName] = headerValue

			// Handle Host header specially for URL
			if strings.ToLower(headerName) == "host" && requestObject.Url != nil {
				// Set the host in the URL if not already set
				if requestObject.Url.Host == "" {
					requestObject.Url.Host = headerValue

					// If scheme is not set and we have a host, default to http
					if requestObject.Url.Scheme == "" {
						requestObject.Url.Scheme = "http"
					}
				}
			}
		}

		headerIndex++
	}

	// Extract body if present
	if bodyStartIndex > 0 && bodyStartIndex < len(lines) {
		bodyLines := lines[bodyStartIndex:]
		body := strings.Join(bodyLines, "\n") // Use consistent line endings in body

		if body != "" {
			requestObject.Body = body

			// Set Content-Length if missing
			if _, exists := requestObject.Headers["Content-Length"]; !exists {
				requestObject.Headers["Content-Length"] = strconv.Itoa(len(body))
			}
		}
	}

	// Handle Content-Type for POST/PUT methods if missing
	if (requestObject.Method == "POST" || requestObject.Method == "PUT") &&
		requestObject.Body != "" {
		if _, exists := requestObject.Headers["Content-Type"]; !exists {
			// Default content type for POST/PUT with body
			requestObject.Headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}

	return requestObject, nil
}

// =====
func parseHTTPBlockResponses(httpFileContent []HTTPFileContent, config *Config) ([]HTTPFileContent, error) {
	// 1. It iterates through each HTTPFileContent in the provided slice
	// 2. For each file, it loops through all HTTPBlocks
	// 3. It identifies if a block is an HTTP request by checking its first line
	// 4. When a request is found, it checks if the next block is an HTTP response
	// 5. If a response is found, it parses it into a structured format using function parseHTTPResponse
	for i := range httpFileContent {
		verbose := config != nil && config.Verbose
		for j := 0; j < len(httpFileContent[i].Blocks); j++ {
			currentBlock := httpFileContent[i].Blocks[j]
			if len(strings.TrimSpace(currentBlock.BlockContent)) == 0 {
				continue
			}
			lines := strings.Split(currentBlock.BlockContent, "\n")
			if len(lines) == 0 {
				continue
			}
			firstLineOfCurrentBlock := strings.TrimSpace(lines[0])
			if !isHTTPRequestLine(firstLineOfCurrentBlock) {
				continue
			}
			if j+1 < len(httpFileContent[i].Blocks) {
				nextBlock := httpFileContent[i].Blocks[j+1]
				nextBlockLines := strings.Split(nextBlock.BlockContent, "\n")

				if len(nextBlockLines) == 0 {
					continue
				}

				firstLineOfNextBlock := strings.TrimSpace(nextBlockLines[0])
				// 5.
				if isHTTPResponseLine(firstLineOfNextBlock) {
					response, err := parseHTTPResponse(nextBlock.BlockContent)
					if err != nil {
						if verbose {
							log.Printf("Error parsing HTTP response: %v", err)
						}
						continue
					}
					expectedResponse := &http.Response{
						StatusCode: response.StatusCode,
						Proto:      response.Protocol,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(response.ResponseBody)),
					}

					for key, value := range response.ResponseHeaders {
						expectedResponse.Header.Set(key, value)
					}
					httpFileContent[i].Blocks[j].ExpectedResponse = expectedResponse
					httpFileContent[i].Blocks[j].ExpectedResponseString = nextBlock.BlockContent
					if verbose {
						log.Printf("Found response for request %d in file %s", j, httpFileContent[i].FilePath)
					}
				}
			}
		}
	}

	return httpFileContent, nil
}

// parseHTTPResponse parses a raw HTTP response string into a structured HTTPResponse
func parseHTTPResponse(responseContent string) (*HTTPResponse, error) {
	// Parse first line to get Protocol, StatusCode, StatusText
	// Pattern: <protocol> <status-code> <status-text>
	// Process subsequent lines as headers until empty line
	// Parse header lines (key: value)
	// Join remaining lines as response body
	lines := strings.Split(responseContent, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty response content")
	}
	firstLine := strings.TrimSpace(lines[0])
	parts := strings.SplitN(firstLine, " ", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid status line format: %s", firstLine)
	}
	protocol := parts[0]
	statusCode, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %s", parts[1])
	}
	statusText := parts[2]
	headers := make(map[string]string)
	bodyStartIndex := 1 // Default to start after status line

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			bodyStartIndex = i + 1
			break
		}

		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) == 2 {
			key := strings.TrimSpace(headerParts[0])
			value := strings.TrimSpace(headerParts[1])
			headers[key] = value
		}
	}
	var bodyBuilder strings.Builder
	for i := bodyStartIndex; i < len(lines); i++ {
		bodyBuilder.WriteString(lines[i])
		if i < len(lines)-1 {
			bodyBuilder.WriteString("\n")
		}
	}
	responseBody := bodyBuilder.String()
	return &HTTPResponse{
		Protocol:        protocol,
		StatusCode:      statusCode,
		StatusText:      statusText,
		ResponseHeaders: headers,
		ResponseBody:    responseBody,
	}, nil
}
