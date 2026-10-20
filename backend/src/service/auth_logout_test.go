package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"offer-hub/backend/src/config"
	"strings"
	"testing"
	"time"
)

type logoutRepo struct {
	blacklisted       bool
	readErr, writeErr error
	token             string
	ttl               time.Duration
	onRead            func()
}

func (r *logoutRepo) IsTokenBlacklisted(context.Context, string) (bool, error) {
	if r.onRead != nil {
		r.onRead()
	}
	return r.blacklisted, r.readErr
}
func (r *logoutRepo) BlacklistToken(_ context.Context, token string, ttl time.Duration) (bool, error) {
	r.token = token
	r.ttl = ttl
	if r.writeErr != nil {
		return false, r.writeErr
	}
	if r.blacklisted {
		return false, nil
	}
	r.blacklisted = true
	return true, nil
}
func logoutTestToken(t *testing.T, header string, claims map[string]any, secret string) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	unsigned := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func TestLogoutBlacklistsForRemainingLifetimeAndRejectsReuse(t *testing.T) {
	cfg := config.JWTConfig{Secret: "test-secret", Expire: 2}
	token, err := NewLoginService(nil, cfg).signToken("user1")
	if err != nil {
		t.Fatal(err)
	}
	repo := &logoutRepo{}
	svc := NewLogoutService(repo, cfg)
	before := time.Now()
	if err := svc.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	after := time.Now()
	claims, err := svc.parseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	expires := time.Unix(claims.ExpiresAt, 0)
	if repo.token != token || repo.ttl <= 0 || repo.ttl < expires.Sub(after) || repo.ttl > expires.Sub(before) {
		t.Fatalf("incorrect blacklist token or ttl: %v", repo.ttl)
	}
	if _, err := svc.AuthenticateToken(context.Background(), token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("reused token authenticated: %v", err)
	}
	if err := svc.Logout(context.Background(), token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("repeated logout: %v", err)
	}
}
func TestLogoutRejectsInvalidTokensBeforeWriting(t *testing.T) {
	now := time.Now().Unix()
	cases := []struct {
		name, header, secret string
		claims               map[string]any
	}{
		{"wrong signature", `{"alg":"HS256"}`, "wrong", map[string]any{"user_id": "u", "exp": now + 3600}},
		{"none algorithm", `{"alg":"none"}`, "secret", map[string]any{"user_id": "u", "exp": now + 3600}},
		{"expired", `{"alg":"HS256"}`, "secret", map[string]any{"user_id": "u", "exp": now - 1}},
		{"no expiry", `{"alg":"HS256"}`, "secret", map[string]any{"user_id": "u"}},
		{"no user", `{"alg":"HS256"}`, "secret", map[string]any{"exp": now + 3600}},
		{"future nbf", `{"alg":"HS256"}`, "secret", map[string]any{"user_id": "u", "exp": now + 3600, "nbf": now + 100}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &logoutRepo{}
			token := logoutTestToken(t, tc.header, tc.claims, tc.secret)
			if err := NewLogoutService(repo, config.JWTConfig{Secret: "secret"}).Logout(context.Background(), token); !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("unexpected error %v", err)
			}
			if repo.token != "" {
				t.Fatal("invalid token written")
			}
		})
	}
	for _, token := range []string{"", "broken", "a.b.c"} {
		if err := NewLogoutService(&logoutRepo{}, config.JWTConfig{Secret: "secret"}).Logout(context.Background(), token); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("malformed accepted: %v", err)
		}
	}
}
func TestLogoutRedisFailureAndExpiryDuringCheck(t *testing.T) {
	failure := errors.New("redis offline")
	cfg := config.JWTConfig{Secret: "secret"}
	token := logoutTestToken(t, `{"alg":"HS256"}`, map[string]any{"user_id": "u", "exp": time.Now().Unix() + 3600}, cfg.Secret)
	for _, repo := range []*logoutRepo{{readErr: failure}, {writeErr: failure}} {
		if err := NewLogoutService(repo, cfg).Logout(context.Background(), token); !errors.Is(err, failure) {
			t.Fatalf("redis failure hidden: %v", err)
		}
	}
	repo := &logoutRepo{onRead: func() { time.Sleep(1100 * time.Millisecond) }}
	token = logoutTestToken(t, `{"alg":"HS256"}`, map[string]any{"user_id": "u", "exp": time.Now().Unix() + 1}, cfg.Secret)
	if err := NewLogoutService(repo, cfg).Logout(context.Background(), token); !errors.Is(err, ErrUnauthenticated) || repo.token != "" {
		t.Fatalf("expired token was written: %v", err)
	}
}

func TestLogoutRejectsAlternateSignatureEncoding(t *testing.T) {
	cfg := config.JWTConfig{Secret: "secret"}
	token := logoutTestToken(t, `{"alg":"HS256"}`, map[string]any{"user_id": "u", "exp": time.Now().Unix() + 3600}, cfg.Secret)
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	// SHA-256 签名的最后一个 Base64 字符有两个未使用位。
	index := strings.IndexByte(alphabet, token[len(token)-1])
	alias := token[:len(token)-1] + string(alphabet[index|1])
	repo := &logoutRepo{}
	if err := NewLogoutService(repo, cfg).Logout(context.Background(), alias); !errors.Is(err, ErrUnauthenticated) || repo.token != "" {
		t.Fatalf("alternate encoding accepted: %v", err)
	}
}
