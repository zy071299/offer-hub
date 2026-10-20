package router

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserInfoRouteRequiresIdentityHeader(t *testing.T) {
	w := httptest.NewRecorder()
	New().ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/user_info/get_user_info", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":401`) {
		t.Fatalf("unexpected response %d %s", w.Code, w.Body)
	}
}
