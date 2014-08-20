package auth

import (
	"testing"
)

func TestHeaderParsing(t *testing.T) {
	// Basic admin:password
	authorization := "Basic YWRtaW46cGFzc3dvcmQ="

	auth, err := parseAuthHeader(authorization)
	if err != nil {
		t.Error(err)
	}

	if auth.Name != "admin" {
		t.Errorf("Detected name does not match: '%s'", auth.Name)
	}
	if auth.Pass != "password" {
		t.Errorf("Detected password does not match: '%s'", auth.Pass)
	}
}
