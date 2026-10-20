package tools

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/config"
	"offer-hub/backend/src/service"
)

type middlewareBlacklist struct {
	blocked   bool
	err       error
	readToken string
	writes    int
}

func (r *middlewareBlacklist) IsTokenBlacklisted(_ context.Context, token string) (bool, error) {
	r.readToken = token
	return r.blocked, r.err
}
func (r *middlewareBlacklist) BlacklistToken(context.Context, string, time.Duration) (bool, error) {
	r.writes++
	return false, errors.New("middleware must not write blacklist")
}
func middlewareTestToken(t *testing.T, secret string, expiry int64, algorithm, userID string) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": algorithm, "typ": "JWT"})
	payload, _ := json.Marshal(map[string]any{"user_id": userID, "exp": expiry})
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestJWTAuthMiddlewareValidatesTokenAndOverwritesIdentity(t *testing.T) {
	secret := "middleware-test-secret"
	expiry := time.Now().Unix() + 3600
	valid := middlewareTestToken(t, secret, expiry, "HS256", "trusted-user")
	for _, tc := range []struct {
		name     string
		headers  []string
		blocked  bool
		storeErr error
		code     int
	}{
		{"success", []string{"Bearer " + valid}, false, nil, 0},
		{"case insensitive scheme", []string{"bearer " + valid}, false, nil, 0},
		{"missing", nil, false, nil, 401},
		{"empty", []string{""}, false, nil, 401},
		{"wrong scheme", []string{"Basic " + valid}, false, nil, 401},
		{"missing token", []string{"Bearer"}, false, nil, 401},
		{"extra token", []string{"Bearer a b"}, false, nil, 401},
		{"duplicate headers", []string{"Bearer " + valid, "Bearer " + valid}, false, nil, 401},
		{"malformed", []string{"Bearer invalid.jwt"}, false, nil, 401},
		{"wrong signature", []string{"Bearer " + middlewareTestToken(t, "wrong", expiry, "HS256", "trusted-user")}, false, nil, 401},
		{"wrong algorithm", []string{"Bearer " + middlewareTestToken(t, secret, expiry, "none", "trusted-user")}, false, nil, 401},
		{"expired", []string{"Bearer " + middlewareTestToken(t, secret, time.Now().Unix()-1, "HS256", "trusted-user")}, false, nil, 401},
		{"no user_id", []string{"Bearer " + middlewareTestToken(t, secret, expiry, "HS256", "")}, false, nil, 401},
		{"blacklisted", []string{"Bearer " + valid}, true, nil, 401},
		{"Redis failure", []string{"Bearer " + valid}, false, errors.New("private Redis error"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &middlewareBlacklist{blocked: tc.blocked, err: tc.storeErr}
			r := gin.New()
			called := false
			r.GET("/protected", jwtAuthMiddleware(service.NewLogoutService(repo, config.JWTConfig{Secret: secret})), func(c *gin.Context) {
				called = true
				if c.GetHeader("user_id") != "trusted-user" || len(c.Request.Header.Values("user_id")) != 1 {
					t.Fatal("untrusted identity reached handler")
				}
				c.JSON(200, gin.H{"code": 0, "user_id": c.GetHeader("user_id")})
			})
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Add("user_id", "forged-user")
			req.Header.Add("user_id", "second-forged-user")
			for _, header := range tc.headers {
				req.Header.Add("Authorization", header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var result struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data any    `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || result.Code != tc.code || called != (tc.code == 0) {
				t.Fatalf("unexpected response %d %s called=%v", w.Code, w.Body, called)
			}
			if tc.code == 401 && (result.Msg != "未认证" || result.Data != nil) {
				t.Fatalf("invalid unauthorized response %s", w.Body)
			}
			if tc.code != 0 && req.Header.Get("user_id") != "" {
				t.Fatal("forged user_id survived rejection")
			}
			if repo.writes != 0 {
				t.Fatal("middleware changed blacklist TTL")
			}
			if (tc.code == 0 || tc.blocked) && repo.readToken != valid {
				t.Fatal("blacklist was not checked for raw token")
			}
		})
	}
}

func TestJWTAuthMiddlewareReadsConfiguredSecret(t *testing.T) {
	old := config.Conf
	t.Cleanup(func() { config.Conf = old })
	config.Conf.JWT.Secret = "configured-secret"
	r := gin.New()
	r.GET("/protected", JWTAuthMiddleware(), func(c *gin.Context) { t.Fatal("request reached handler without Redis verification") })
	for _, tc := range []struct {
		secret string
		code   int
	}{
		{"wrong-secret", 401},
		// 正确签名应通过 JWT 校验，进入尚未初始化的 Redis 并返回 500。
		{"configured-secret", 500},
	} {
		token := middlewareTestToken(t, tc.secret, time.Now().Unix()+3600, "HS256", "user")
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var result struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Code != tc.code {
			t.Fatalf("configured secret was not used: %s", w.Body)
		}
	}
}
