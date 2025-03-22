package main

import (
	"testing"
)

func TestGetRawContent(t *testing.T) {
	expectedRawContent := "GET http://localhost:8080/v1/comments/1 HTTP/1.1"
	config := &Config{
		WatchFolderPath: "",
		WatchFilePath:   "",
		HTTPFilePath:    "./http_folder/test_0.http",
		HTTPFolderPath:  "",
		ExcludeFile:     "",
		ExcludeFolder:   "",
		WaitRequestTime: 1000, // Default 1 second in milliseconds
		Verbose:         false,
	}

	httpFileContent, err := getRawContent(config)
	if err != nil {
		t.Fatalf("Failed function httpFileContent: %v", err)
	}
	if len(httpFileContent) == 0 {
		t.Fatal("No http file content returned")
	}

	if httpFileContent[0].RawContent != expectedRawContent {
		t.Fatalf("Unexpected raw content.\nExpected: %q\nGot: %q", expectedRawContent, httpFileContent[0].RawContent)

	}
}

//	type HTTPFileContent struct {
//		RawContent      string
//		FilePath        string
//		GlobalVariables map[string]string
//		Blocks          []HTTPBlock
//	}
func TestRemoveComments(t *testing.T) {
	httpFileContent := []HTTPFileContent{
		{
			RawContent: "GET http://localhost:8080/v1/comments/1 HTTP/1.1",
		},
		{
			RawContent: "GET http://localhost:8080/v1/comments/1 HTTP/1.1\n//a comment",
		},
		{
			RawContent: "GET http://localhost:8080/v1/comments/1 HTTP/1.1\n# a comment",
		},
	}
	httpFileContent, err := removeComments(httpFileContent)
	if err != nil {
		t.Fatalf("removeComments return error: %v", err)
	}
	expectedRawContent := "GET http://localhost:8080/v1/comments/1 HTTP/1.1"
	if httpFileContent[0].RawContent != expectedRawContent {
		t.Fatalf("Expected raw content: %q got %q", httpFileContent[0].RawContent, expectedRawContent)
	}

	expectedRawContent2 := "GET http://localhost:8080/v1/comments/1 HTTP/1.1"
	if httpFileContent[1].RawContent != expectedRawContent2 {
		t.Fatalf("Expected raw content: %q got %q", httpFileContent[0].RawContent, expectedRawContent2)
	}
	expectedRawContent3 := "GET http://localhost:8080/v1/comments/1 HTTP/1.1\n# a comment"
	if httpFileContent[2].RawContent != expectedRawContent3 {
		t.Fatalf("Expected raw content: %q got %q", httpFileContent[0].RawContent, expectedRawContent3)
	}
}
