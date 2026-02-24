package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSumHandler(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantResult float64
		wantError  bool
	}{
		{"basic sum", "?a=2&b=3", http.StatusOK, 5, false},
		{"negative numbers", "?a=-10&b=5", http.StatusOK, -5, false},
		{"decimals", "?a=1.5&b=2.5", http.StatusOK, 4, false},
		{"zeros", "?a=0&b=0", http.StatusOK, 0, false},
		{"large numbers", "?a=1000000&b=2000000", http.StatusOK, 3000000, false},
		{"missing both params", "", http.StatusBadRequest, 0, true},
		{"missing b", "?a=1", http.StatusBadRequest, 0, true},
		{"missing a", "?b=1", http.StatusBadRequest, 0, true},
		{"invalid a", "?a=abc&b=1", http.StatusBadRequest, 0, true},
		{"invalid b", "?a=1&b=abc", http.StatusBadRequest, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/sum"+tt.query, nil)
			w := httptest.NewRecorder()

			sumHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if !tt.wantError {
				var resp SumResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Result != tt.wantResult {
					t.Errorf("result = %f, want %f", resp.Result, tt.wantResult)
				}
			} else {
				var resp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if resp.Error == "" {
					t.Error("expected non-empty error message")
				}
			}
		})
	}
}
