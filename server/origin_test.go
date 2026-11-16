package main

import (
	"net/http/httptest"
	"testing"
)

func TestDeploymentOrigins(t *testing.T) {
	cases := []struct {
		origin  string
		allowed bool
	}{
		{"https://iron-table.netlify.app", true},
		{"https://preview.example.com", true},
		{"https://other.netlify.app", false},
		{"http://iron-table.netlify.app", false},
		{"https://iron-table.netlify.app.attacker.com", false},
		{"null", false}, {"", true}, {"https://server.onrender.com", true},
	}
	for _, tc := range cases {
		r := httptest.NewRequest("GET", "https://server.onrender.com/ws", nil)
		r.Header.Set("Origin", tc.origin)
		if originAllowed(r, "https://iron-table.netlify.app, https://preview.example.com") != tc.allowed {
			t.Errorf("Incorrect origin policy for %q", tc.origin)
		}
	}
}
