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

			blocksLength := len(httpFileContent[0].Blocks)
			blocksLength_Expected := len(testCase.httpExpected)
			// fmt.Println(blocksLength)
			// fmt.Println(blocksLength_Expected)
			if blocksLength != blocksLength_Expected {
				t.Errorf("Incorrect length. expected: %d, Got: %d", blocksLength, blocksLength_Expected)
				return
			}

			for i, block := range httpFileContent[0].Blocks {

				// =======
				//   HTTP Method
				// =======
				if block.Request == nil {
					t.Errorf("\033[33merror nil at httpFileContent[i].Blocks[%d]\033[0m\n", i)
					return
				}
				// HTTPMethod := block.Request.Method
				// HTTPMethod_Expected := testCase.httpExpected[i].Method
				//if HTTPMethod == "" {
				//	t.Errorf("No HTTPMethod[%d], Exp: %s", i, HTTPMethod_Expected)
				//}
				// if HTTPMethod != HTTPMethod_Expected {
				// 	t.Errorf("Incorrect HTTP method [%d]. expected: %s, Got: %s", i, HTTPMethod, HTTPMethod_Expected)
				// 	return
				// }
				// =======
				//   URL
				// =======
				// URL := block.Request.URL.String()
				// URL_Expected := testCase.httpExpected[i].Url
				// if URL != URL_Expected {
				// 	t.Errorf("Incorrect URL [%d]. expected: %s, Got: %s", i, URL, URL_Expected)
				// 	return
				// }
			}

		})
	}
}
