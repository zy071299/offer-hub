package ctrl

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"offer-hub/backend/src/service"
	"testing"
)

type logoutService struct {
	err   error
	token string
}

func (s *logoutService) Logout(_ context.Context, token string) error { s.token = token; return s.err }
func TestLogoutResponseAndAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name, header string
		err          error
		code         int
	}{
		{"success", "Bearer valid-token", nil, 0},
		{"missing", "", nil, 401}, {"wrong scheme", "Basic valid-token", nil, 401},
		{"no token", "Bearer", nil, 401}, {"extra token", "Bearer a b", nil, 401},
		{"invalid JWT", "Bearer invalid-token", service.ErrUnauthenticated, 401},
		{"redis failure", "Bearer valid-token", errors.New("redis unavailable"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &logoutService{err: tc.err}
			r := gin.New()
			r.POST("/auth/logout", NewLogoutController(svc).Logout)
			req := httptest.NewRequest("POST", "/auth/logout", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var result struct {
				Code int               `json:"code"`
				Msg  string            `json:"msg"`
				Data map[string]string `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || result.Code != tc.code {
				t.Fatalf("unexpected response %d %s", w.Code, w.Body)
			}
			if tc.code == 0 {
				if result.Msg != "success" || result.Data["message"] != "登出成功" || svc.token != "valid-token" {
					t.Fatalf("incorrect success %s", w.Body)
				}
			} else if result.Data != nil {
				t.Fatal("error leaked data")
			}
		})
	}
}
