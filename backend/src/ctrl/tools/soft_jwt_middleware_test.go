package tools

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/config"
	"offer-hub/backend/src/service"
)

func TestSoftJWTAuthMiddlewareAlwaysContinuesWithoutRedis(t *testing.T) {
	secret := "soft-test-secret"
	expiry := time.Now().Unix() + 3600
	valid := middlewareTestToken(t, secret, expiry, "HS256", "trusted-user")
	for _, tc := range []struct {
		name    string
		headers []string
		wantID  string
	}{
		{"missing", nil, ""},
		{"empty", []string{""}, ""},
		{"valid", []string{"Bearer " + valid}, "trusted-user"},
		{"lowercase scheme", []string{"bearer " + valid}, "trusted-user"},
		{"duplicate headers", []string{"Bearer " + valid, "Bearer " + valid}, ""},
		{"wrong scheme", []string{"Basic " + valid}, ""},
		{"missing token", []string{"Bearer"}, ""},
		{"extra token", []string{"Bearer a b"}, ""},
		{"malformed", []string{"Bearer invalid.jwt"}, ""},
		{"expired", []string{"Bearer " + middlewareTestToken(t, secret, time.Now().Unix()-1, "HS256", "trusted-user")}, ""},
		{"bad signature", []string{"Bearer " + middlewareTestToken(t, "wrong-secret", expiry, "HS256", "trusted-user")}, ""},
		{"wrong algorithm", []string{"Bearer " + middlewareTestToken(t, secret, expiry, "none", "trusted-user")}, ""},
		{"missing identity", []string{"Bearer " + middlewareTestToken(t, secret, expiry, "HS256", "")}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// 即使 token 已拉黑或 Redis 不可用，也不能访问 Redis。
			repo := &middlewareBlacklist{blocked: true, err: errors.New("Redis unavailable")}
			parser := service.NewLogoutService(repo, config.JWTConfig{Secret: secret})
			r := gin.New()
			calls := 0
			r.GET("/public", softJWTAuthMiddleware(parser), func(c *gin.Context) {
				calls++
				if c.GetHeader("user_id") != tc.wantID {
					t.Fatalf("identity = %q want %q", c.GetHeader("user_id"), tc.wantID)
				}
				c.String(202, "handler response")
			})
			req := httptest.NewRequest("GET", "/public", nil)
			req.Header.Add("user_id", "forged-user")
			req.Header.Add("user_id", "second-forged-user")
			for _, header := range tc.headers {
				req.Header.Add("Authorization", header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if calls != 1 || w.Code != 202 || w.Body.String() != "handler response" {
				t.Fatalf("middleware interrupted handler: %d %s calls=%d", w.Code, w.Body, calls)
			}
			if repo.readToken != "" || repo.writes != 0 {
				t.Fatal("soft authentication accessed Redis")
			}
		})
	}
}

func TestSoftJWTAuthMiddlewareUsesConfiguredSecret(t *testing.T) {
	old := config.Conf
	t.Cleanup(func() { config.Conf = old })
	for _, secret := range []string{"configured-soft-secret", ""} {
		config.Conf.JWT.Secret = secret
		token := middlewareTestToken(t, "configured-soft-secret", time.Now().Unix()+3600, "HS256", "trusted-user")
		r := gin.New()
		called := false
		r.GET("/public", SoftJWTAuthMiddleware(), func(c *gin.Context) {
			called = true
			want := ""
			if secret != "" {
				want = "trusted-user"
			}
			if c.GetHeader("user_id") != want {
				t.Fatalf("identity = %q want %q", c.GetHeader("user_id"), want)
			}
			c.Status(204)
		})
		req := httptest.NewRequest("GET", "/public", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if !called || w.Code != 204 {
			t.Fatalf("soft auth blocked request: %d", w.Code)
		}
	}
}
