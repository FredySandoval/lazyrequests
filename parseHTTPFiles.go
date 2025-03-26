package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

//	type rawHTTPFileContent struct {
//	    Content  string
//	    FilePath string
//	}
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
	httpFileContent, err = parseHTTPBlockRequests(httpFileContent)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - parseHTTPBlockRequests() %w", err)
	}
	httpFileContent, err = parseHTTPBlockResponses(httpFileContent, config)
	if err != nil {
		return nil, fmt.Errorf("error at parseHTTPFiles.go - parseHTTPBlockResponse() %w", err)
	}

	// TODO - create function to remove the RESPONSES from the blocks
	// maybe start working on the testing.

	// fmt.Printf("Hex (spaced):\r\n% x\n", []byte(rawRequest))
	return httpFileContent, nil
}
func parseHTTPBlockResponses(httpFileContent []HTTPFileContent, config *Config) ([]HTTPFileContent, error) {
	logVerbose(config, "Processing HTTP response blocks and attaching them to requests...")
	for i := range httpFileContent {
		file := &httpFileContent[i]
		logVerbose(config, fmt.Sprintf("Processing file: %s", file.FilePath))

		// Iterate through blocks, but stop at the last block (since we need to look ahead)
		for j := 0; j < len(file.Blocks)-1; j++ {
			currentBlock := &file.Blocks[j]
			nextBlock := &file.Blocks[j+1]

			// Skip empty current blocks
			if strings.TrimSpace(currentBlock.BlockContent) == "" {
				logVerbose(config, fmt.Sprintf("Skipping empty block %d", currentBlock.ID))
				continue
			}

			// Skip empty next blocks
			if strings.TrimSpace(nextBlock.BlockContent) == "" {
				logVerbose(config, fmt.Sprintf("Next block %d is empty, skipping", nextBlock.ID))
				continue
			}

			// Check if current block is a request
			currentBlockLines := strings.Split(strings.TrimSpace(currentBlock.BlockContent), "\n")
			if len(currentBlockLines) == 0 {
				continue
			}

			firstLineOfCurrentBlock := currentBlockLines[0]
			if !isHTTPRequestLine(firstLineOfCurrentBlock) {
				logVerbose(config, fmt.Sprintf("Current block %d is not a request block, skipping", currentBlock.ID))
				continue // Skip if current block is not a request
			}

			// Check if the next block is a response
			nextBlockLines := strings.Split(strings.TrimSpace(nextBlock.BlockContent), "\n")
			if len(nextBlockLines) == 0 {
				continue
			}

			firstLineOfNextBlock := nextBlockLines[0]
			if !isHTTPResponseLine(firstLineOfNextBlock) {
				logVerbose(config, fmt.Sprintf("Next block %d is not a response block, skipping", nextBlock.ID))
				continue // Skip if next block is not a response
			}

			logVerbose(config, fmt.Sprintf("Processing response block %d for request block %d", nextBlock.ID, currentBlock.ID))

			// Create a buffer from the next block content
			bytesBuffer := bytes.NewBufferString(nextBlock.BlockContent)
			bufferReader := bufio.NewReader(bytesBuffer)

			// Parse the HTTP response
			resp, err := http.ReadResponse(bufferReader, nil)
			if err != nil {
				return nil, fmt.Errorf("error parsing HTTP response in file %s, block %d: %w",
					file.FilePath, nextBlock.ID, err)
			}

			// Read the response body if it exists
			var bodyBytes []byte
			if resp.Body != nil {
				bodyBytes, err = io.ReadAll(resp.Body)
				if err != nil {
					return nil, fmt.Errorf("error reading response body in file %s, block %d: %w",
						file.FilePath, nextBlock.ID, err)
				}
				// Close the body reader
				resp.Body.Close()

				// Create a new body reader for later use
				resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// Keep a string representation for debugging or display purposes
			var responseStr strings.Builder

			// Format the response for the string representation
			responseStr.WriteString(fmt.Sprintf("HTTP/%d.%d %d %s\r\n",
				resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status))

			// Add headers
			for key, values := range resp.Header {
				for _, value := range values {
					responseStr.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
				}
			}

			// Add an empty line to separate headers from body
			responseStr.WriteString("\r\n")

			// Add body if exists
			if len(bodyBytes) > 0 {
				responseStr.Write(bodyBytes)
			}

			// Attach the response to the current block
			currentBlock.ExpectedResponse = resp
			currentBlock.ExpectedResponseString = responseStr.String()

			logVerbose(config, fmt.Sprintf("Attached response from block %d to request block %d with status code %d",
				nextBlock.ID, currentBlock.ID, resp.StatusCode))
		}
	}

	logVerbose(config, "Completed processing HTTP response blocks and attaching them to requests")
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
func parseHTTPBlockRequests(httpFileContent []HTTPFileContent) ([]HTTPFileContent, error) {
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

			// Create a buffer from the block content
			bytesBuffer := bytes.NewBufferString(block.BlockContent)
			bufferReader := bufio.NewReader(bytesBuffer)

			// Parse the HTTP request
			req, err := http.ReadRequest(bufferReader)
			if err != nil {
				return nil, fmt.Errorf("error parsing HTTP request in file %s, block %d: %w",
					file.FilePath, block.ID, err)
			}

			// Read the request body if it exists
			var bodyBytes []byte
			if req.Body != nil {
				bodyBytes, err = io.ReadAll(req.Body)
				if err != nil {
					return nil, fmt.Errorf("error reading request body in file %s, block %d: %w",
						file.FilePath, block.ID, err)
				}
				// Close the body reader
				req.Body.Close()

				// Create a new body reader for later use
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// Create a new request that can be sent later
			newReq, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewBuffer(bodyBytes))
			if err != nil {
				return nil, fmt.Errorf("error creating new request in file %s, block %d: %w",
					file.FilePath, block.ID, err)
			}

			// Copy headers from the original request
			for key, values := range req.Header {
				for _, value := range values {
					newReq.Header.Add(key, value)
				}
			}

			// Keep a string representation for debugging or display purposes
			var requestStr strings.Builder

			// Format the request for the string representation
			requestStr.WriteString(fmt.Sprintf("%s %s HTTP/%d.%d\r\n",
				newReq.Method, newReq.URL.String(), req.ProtoMajor, req.ProtoMinor))

			// Add headers
			for key, values := range newReq.Header {
				for _, value := range values {
					requestStr.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
				}
			}

			// Add an empty line to separate headers from body
			requestStr.WriteString("\r\n")

			// Add body if exists
			if len(bodyBytes) > 0 {
				requestStr.Write(bodyBytes)
			}

			// Update both the Request object and RequestString fields in the block
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
