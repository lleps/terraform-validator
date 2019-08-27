package main

import "testing"

func TestExtractNameFromPath(t *testing.T) {
	cases := map[string]string {
		"/path/to/my/file": "file",
		"myfile": "myfile",
	}
	for input, expected := range cases {
		got := extractNameFromPath(input)
		if got != expected {
			t.Errorf("TestExtractNameFromPath for '%s': expected: '%s', got: '%s", input, expected, got)
		}
	}
}
