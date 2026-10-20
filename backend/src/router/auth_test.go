package router

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogoutRouteRequiresLogin(t *testing.T) {
	w := httptest.NewRecorder()
	New().ServeHTTP(w, httptest.NewRequest("POST", "/auth/logout", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":401`) {
		t.Fatalf("unexpected response %d %s", w.Code, w.Body)
	}
}

func TestRegisterRouteAllowsGuests(t *testing.T) {
	w := httptest.NewRecorder()
	New().ServeHTTP(w, httptest.NewRequest("POST", "/auth/register", strings.NewReader(`{}`)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":400`) {
		t.Fatalf("unexpected response %d %s", w.Code, w.Body)
	}
}

func TestLoginRouteAllowsGuests(t *testing.T) {
	w := httptest.NewRecorder()
	New().ServeHTTP(w, httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{}`)))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":400`) {
		t.Fatalf("unexpected response %d %s", w.Code, w.Body)
	}
}
