package main

import (
	testcases "lazyrequests/http_folder"
	"path/filepath"
	"testing"
)

func TestProcessHTTPFiles(t *testing.T) {
	// Define test file pairs
	testCases := []struct {
		httpFile     string
		httpExpected []testcases.RequestInfo
	}{
		{"test_0_bare.http", testcases.Test_0_bare},
		{"test_1_comments.http", testcases.Test_1_comments},
		{"test_2_variables_basic.http", testcases.Test_2_variables_basic},
		{"test_3_parse_blocks.http", testcases.Test_3_parse_blocks},
		{"test_4_parse_requests.http", testcases.Test_4_parse_requests},
		{"test_5_parse_responses.http", testcases.Test_5_parse_responses},
	}

	// Loop through all test files
	for _, testCase := range testCases {
		t.Run(testCase.httpFile, func(t *testing.T) {
			filepath_original := filepath.Join("./http_folder", testCase.httpFile)

			config := &Config{
				WatchFolderPath: "",
				WatchFilePath:   "",
				HTTPFilePath:    filepath_original,
				HTTPFolderPath:  "",
				ExcludeFile:     "",
				ExcludeFolder:   "",
				WaitRequestTime: 1000, // Default 1 second in milliseconds
				Verbose:         false,
			}

			httpFileContent, err := processHTTPFiles(config)
			if err != nil {
				t.Errorf("got error on function processHTTPFiles: %v", err)
				return
			}

			// LENGTH
			blocksLength := len(httpFileContent[0].Blocks)
			blocksLength_Expected := len(testCase.httpExpected)
			if blocksLength != blocksLength_Expected {
				t.Errorf("Incorrect length. expected: %d, Got: %d", blocksLength, blocksLength_Expected)
				return
			}

			for i, block := range httpFileContent[0].Blocks {

				// =======
				//   HTTP Method
				// =======
				// if block.Request == nil {
				// 	t.Errorf("\033[33merror nil at httpFileContent[i].Blocks[%d]\033[0m\n", i)
				// 	return
				// }
				HTTPMethod := block.Request.Method
				HTTPMethod_Expected := testCase.httpExpected[i].Method
				if HTTPMethod == "" {
					t.Errorf("No HTTPMethod[%d], Exp: %s", i, HTTPMethod_Expected)
				}
				if HTTPMethod != HTTPMethod_Expected {
					t.Errorf("Incorrect HTTP method [%d]. expected: %s, Got: %s", i, HTTPMethod, HTTPMethod_Expected)
					return
				}
				// =======
				//   URL
				// =======
				URL := block.Request.Url
				URL_Expected := testCase.httpExpected[i].Url
				if URL != URL_Expected {
					t.Errorf("Incorrect URL [%d].\nexpected: %s\nGot:      %s", i, URL, URL_Expected)
					return
				}

				CommentIdentifier := block.CommentIdentifier
				CommentIdentifier_Expected := testCase.httpExpected[i].CommentIdentifier
				if CommentIdentifier != CommentIdentifier_Expected {
					t.Errorf("Incorrect COMMENT [%d].\nexpected: %s\nGot:      %s", i, CommentIdentifier, CommentIdentifier_Expected)
					return
				}

				// ====
				// Request body
				// ====
				bodyString := block.Request.Body

				expectedBody := testCase.httpExpected[i].Body
				if bodyString != expectedBody {
					t.Errorf("Incorrect Body [%d].\nexpected: %x\nGot:      %x", i, []byte(expectedBody), []byte(bodyString))
					return
				}
				//fmt.Printf("Hex (spaced):\r\n% x\n", []byte(bodyString))
				got := block.Request.Headers["User-Agent"]
				if got != "" {
					if block.Request.Headers["User-Agent"] != "curl/8.6.0" {
						t.Errorf("Incorrect Body [%d].\nexpected: %s\nGot:      %s", i, "[curl/8.6.0]", block.Request.Headers["User-Agent"])
						return
					}
				}
				// ===
				// response
				// ===
				res := block.ExpectedResponse
				res_expected := testCase.httpExpected[i]

				if res != nil {
					if res.Status != res_expected.Status {
						t.Errorf("Incorrect Status [%d].\nexpected: %s\nGot:      %s", i, res_expected.Status, res.Status)
						return
					}
					if res.StatusCode != res_expected.StatusCode {
						t.Errorf("Incorrect Status [%d].\nexpected: %d\nGot:      %d", i, res_expected.StatusCode, res.StatusCode)
						return
					}
				}
			}

		})
	}
}
