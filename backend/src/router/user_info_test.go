package router

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserInfoRouteRequiresLogin(t *testing.T) {
	w := httptest.NewRecorder()
	New().ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/user_info/get_user_info", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":401`) {
		t.Fatalf("unexpected response %d %s", w.Code, w.Body)
	}
}

func TestUserInfoRouteRejectsForgedIdentity(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/user_info/get_user_info", nil)
	req.Header.Set("user_id", "forged-user")
	w := httptest.NewRecorder()
	New().ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":401`) || req.Header.Get("user_id") != "" {
		t.Fatalf("forged identity accepted: %d %s", w.Code, w.Body)
	}
}
