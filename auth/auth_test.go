package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth(t *testing.T) {
	apiKeyToUserName := map[string]string{
		"test-api-key-1": "user-1",
		"test-api-key-2": "user-2",
	}
	tests := []struct {
		name           string
		req            func() *http.Request
		expectedStatus int
		expectedUser   string
	}{
		{
			name:           "no auth header returns 401",
			req:            func() *http.Request { return httptest.NewRequest("GET", "/", nil) },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "auth header not in map returns 401",
			req: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Authorization", "Bearer not-in-map")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "auth header in map returns 200",
			req: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Authorization", "Bearer test-api-key-1")
				return req
			},
			expectedStatus: http.StatusOK,
			expectedUser:   "user-1",
		},
		{
			name: "auth header doesn't need Bearer prefix",
			req: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Authorization", "test-api-key-2")
				return req
			},
			expectedStatus: http.StatusOK,
			expectedUser:   "user-2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user string
			var ok bool
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, ok = GetUser(r)
				if !ok {
					t.Error("expected user to be set")
				}
				w.WriteHeader(http.StatusOK)
			})

			auth := New(apiKeyToUserName, h)
			w := httptest.NewRecorder()
			auth.ServeHTTP(w, tt.req())
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if user != tt.expectedUser {
				t.Errorf("expected user to be %s, got %s", tt.expectedUser, user)
			}
		})
	}
}
