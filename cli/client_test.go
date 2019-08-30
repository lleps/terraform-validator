package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractNameFromPath(t *testing.T) {
	cases := map[string]string {
		"/path/to/my/file": "file",
		"myfile": "myfile",
	}
	for input, expected := range cases {
		assert.Equal(t, expected, extractNameFromPath(input), "for input: " + input)
	}
}