// This program watches a directory or file for file changes and sends HTTP requests based on a template file, with file extention .http
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
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
	requestCount  int         = 1 // Counter for total requests sent

)

const (
	C_Red       = "\033[31m"
	C_Green     = "\033[32m"
	C_Blue      = "\033[34m"
	C_Reset     = "\033[0m"
	C_White     = "\033[37m"
	C_Gray      = "\033[90m"
	C_Yellow    = "\033[33m"
	C_Purple    = "\033[35m"
	C_Cyan      = "\033[36m"
	C_Bold      = "\033[1m"
	C_Underline = "\033[4m"
)

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

	// send requests at start
	sendRequests(httpFileContentParsed, config)

	// Create a new watcher.
	w, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("fsnotify.NewWatcher() err %v", err)
		return
	}
	defer w.Close()

	// Start listening for events.
	go dedupLoop(w, config, httpFileContentParsed)

	// Add all paths from the commandline.

	if config.WatchFilePath != "" {
		err = w.Add(config.WatchFilePath)
	} else {
		err = w.Add(config.WatchFolderPath)
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	<-make(chan struct{}) // Block forever

}

func sendRequests(httpFileContentParsed []HTTPFileContent, config *Config) {
	currentTime := time.Now().Format("03:04 PM")
	fmt.Printf("%s[%d] %s %s\n", C_Underline+C_Bold+C_Cyan, requestCount, currentTime, C_Reset)

	httpRequestTimeOut := time.Duration(config.HTTPRequestTimeout) * time.Millisecond
	waitRequestTime := config.SleepTime * int(time.Millisecond)
	for j, fileContent := range httpFileContentParsed {
		for k, block := range fileContent.Blocks {
			// Add a sleep, to allow server to initialize and in between requests
			time.Sleep(time.Duration(waitRequestTime)) // Fixed 100ms wait between requests

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
			// Start timing the request
			startTime := time.Now()

			client := &http.Client{}
			resp, err := client.Do(newReq)

			// Calculate elapsed time
			elapsedTime := time.Since(startTime)
			elapsedMs := elapsedTime.Milliseconds()

			if err != nil {
				fileName := filepath.Base(fileContent.FilePath)
				logVerbose(config, "Error at main.go: client := &http.Client{}")
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
				if block.CommentIdentifier != "" {
					fmt.Printf("%s%s%s\n", C_Purple, block.CommentIdentifier, C_Reset)
				}
				//fmt.Printf("%10s %4s %s%s%dms %s%s%s\n", C_Bold+C_Blue+resp.Request.Method+C_Reset, C_Green, resp.Status, C_Yellow, elapsedMs, C_Gray, resp.Request.URL, C_Reset)
				fmt.Printf("%s%-6s %s%-12s %s%3dms %s%s\n", C_Bold+C_Blue, resp.Request.Method, C_Reset+C_Green, resp.Status, C_Yellow, elapsedMs, C_Gray, resp.Request.URL)
			} else {
				fmt.Printf("%s[X][ %s ] Expected: [ %s ] Got: [ %s ] %s \n", C_Red, results.MSG, results.Expected, results.Got, C_Reset)
			}
			// LOG IF WINS
			resp.Body.Close()
			cancel()
		}
	}
	fmt.Printf("%sdone.%s\n", C_Gray, C_Reset)
}
func runCmd(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Run()
}
func ClearTerminal() {
	switch runtime.GOOS {
	case "darwin":
		runCmd("clear")
	case "linux":
		runCmd("clear")
	case "windows":
		runCmd("cmd", "/c", "cls")
	default:
		runCmd("clear")
	}
}

func dedupLoop(w *fsnotify.Watcher, config *Config, httpFileContentParsed []HTTPFileContent) {
	var (
		// Wait 100ms for new events; each new event resets the timer.
		waitFor = 100 * time.Millisecond

		// Keep track of the timers, as path → timer.
		mu     sync.Mutex
		timers = make(map[string]*time.Timer)

		// Callback we run.
		printEvent = func(e fsnotify.Event) {
			// reload the the HTTP Files since they've changed
			newHttpFileContentParsed, err := processHTTPFiles(config)
			if err != nil {
				fmt.Printf("%sError reprocessing HTTP files: %v%s\n", C_Red, err, C_Reset)
				return
			}
			// Update the parsed content
			httpFileContentParsed = newHttpFileContentParsed
			// Clear terminal and increase request count
			ClearTerminal()
			requestCount++

			// Send the HTTP requests
			sendRequests(httpFileContentParsed, config)
			// HERE the magic happens
			logVerbose(config, "Watching", e.String())

			// Don't need to remove the timer if you don't have a lot of files.
			mu.Lock()
			delete(timers, e.Name)
			mu.Unlock()
		}
	)

	for {
		select {
		// Read from Errors.
		case err, ok := <-w.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}
			fmt.Println("Error at dedupLoop case !ok", err)
		// Read from Events.
		case e, ok := <-w.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}

			// We just want to watch for file creation, so ignore everything
			// outside of Create and Write.
			if !e.Has(fsnotify.Create) && !e.Has(fsnotify.Write) {
				continue
			}

			// Get timer.
			mu.Lock()
			t, ok := timers[e.Name]
			mu.Unlock()

			// No timer yet, so create one.
			if !ok {
				t = time.AfterFunc(math.MaxInt64, func() { printEvent(e) })
				t.Stop()

				mu.Lock()
				timers[e.Name] = t
				mu.Unlock()
			}

			// Reset the timer for this path, so it will start from 100ms again.
			t.Reset(waitFor)
		}
	}
}
