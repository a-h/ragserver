package main

import "testing"

func TestCreateURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		paths    []string
		expected string
	}{
		{
			name:     "if no paths are provided, the base URL is used",
			baseURL:  "http://localhost",
			paths:    nil,
			expected: "http://localhost",
		},
		{
			name:     "base URLs with trailing slashes are supported",
			baseURL:  "http://localhost/",
			paths:    []string{"a"},
			expected: "http://localhost/a",
		},
		{
			name:     "spaces are URL path encoded",
			baseURL:  "http://localhost",
			paths:    []string{"file 1.txt"},
			expected: "http://localhost/file%201.txt",
		},
		{
			name:     "multiple paths are supported",
			baseURL:  "http://localhost",
			paths:    []string{"a", "b", "c.txt"},
			expected: "http://localhost/a/b/c.txt",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := createURL(test.baseURL, test.paths...)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if actual != test.expected {
				t.Errorf("expected %q, got %q", test.expected, actual)
			}
		})
	}
}

func TestCreateURLError(t *testing.T) {
	baseURL := "://"
	paths := []string{"a", "b", "c.txt"}
	actual, err := createURL(baseURL, paths...)
	if err == nil {
		t.Errorf("expected error, got nil, %q", actual)
	}
}
