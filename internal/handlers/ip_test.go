package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		wantIP     string
	}{
		{
			name:       "remote addr only",
			remoteAddr: "203.0.113.1:12345",
			wantIP:     "203.0.113.1",
		},
		{
			name:       "x-real-ip takes priority",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Real-Ip":       "203.0.113.50",
				"X-Forwarded-For": "198.51.100.1",
			},
			wantIP: "203.0.113.50",
		},
		{
			name:       "x-forwarded-for single entry",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.10",
			},
			wantIP: "203.0.113.10",
		},
		{
			name:       "x-forwarded-for multiple entries uses first",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.10, 10.0.0.2, 10.0.0.3",
			},
			wantIP: "203.0.113.10",
		},
		{
			name:       "x-forwarded-for with whitespace",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "  203.0.113.10 , 10.0.0.2",
			},
			wantIP: "203.0.113.10",
		},
		{
			name:       "x-real-ip with whitespace",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Real-Ip": "  203.0.113.50 ",
			},
			wantIP: "203.0.113.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}

			got := clientIP(r)
			assert.Equal(t, tt.wantIP, got)
		})
	}
}

func TestRawIPHandler(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		wantIP     string
	}{
		{
			name:       "returns ip as plain text",
			remoteAddr: "203.0.113.1:12345",
			wantIP:     "203.0.113.1",
		},
		{
			name:       "respects x-forwarded-for",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.10",
			},
			wantIP: "203.0.113.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			RawIPHandler().ServeHTTP(w, r)

			assert.Equal(t, tt.wantIP, w.Body.String())
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.Equal(t, tt.wantIP, w.Header().Get("X-Incoming-IP"))
		})
	}
}

func TestIPHandler(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		wantIP     string
	}{
		{
			name:       "renders template with remote addr",
			remoteAddr: "203.0.113.1:12345",
			wantIP:     "203.0.113.1",
		},
		{
			name:       "renders template with x-real-ip",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Real-Ip": "203.0.113.50",
			},
			wantIP: "203.0.113.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			IPHandler().ServeHTTP(w, r)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.wantIP, w.Header().Get("X-Incoming-IP"))
			assert.Contains(t, w.Body.String(), tt.wantIP)
		})
	}
}
