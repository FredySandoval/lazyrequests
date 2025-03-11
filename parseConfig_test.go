package main

import (
	"flag"
	"os"
	"testing"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestParseConfig_Errors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args passed", []string{"main"}},
		{"non existing folder", []string{"main", "--watch-folder", "./nonexisting/"}},
		{"non existing file", []string{"main", "--watch-file", "./nonexisting/file.http"}},
		{"file as folder", []string{"main", "--watch-folder", "./main.go"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()
			os.Args = tc.args
			config, err := parseConfig()

			if config != nil {
				t.Errorf("Should be nil: %v", config)
			}
			if err == nil {
				t.Errorf("Should output error: %v", err)
			}
		})
	}
}
