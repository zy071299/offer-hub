package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"offer-hub/backend/src/config"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"strings"
	"testing"
	"time"
)

type loginRepo struct {
	user data.UserInfo
	err  error
}

func (r loginRepo) GetUserByUsername(context.Context, string) (data.UserInfo, error) {
	return r.user, r.err
}

func TestPasswordLoginSignsConfiguredTokenAndReturnsUserInfo(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := data.UserInfo{UserID: "user-123", Username: "cat", Password: string(hash), NickName: "猫", Avatar: "avatar", Sex: 2, VIP: true, Phone: "123", Email: "cat@example.com", UserStatus: 1, UserType: 2}
	cfg := config.JWTConfig{Secret: "test-signing-secret", Expire: 3}
	before := time.Now().Unix()
	result, err := NewLoginService(loginRepo{user: user}, cfg).PasswordLogin(context.Background(), model.PasswordLoginReq{Username: "cat", Password: "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if result.UserInfo != (model.PasswordLoginUserInfo{UserID: "user-123", Username: "cat", NickName: "猫", Avatar: "avatar", Sex: 2, VIP: true, Phone: "123", Email: "cat@example.com", UserStatus: 1, UserType: 2}) {
		t.Fatalf("unexpected user info %+v", result.UserInfo)
	}
	parts := strings.Split(result.Token, ".")
	if len(parts) != 3 {
		t.Fatal("invalid JWT")
	}
	header, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var h map[string]string
	if err := json.Unmarshal(header, &h); err != nil || h["alg"] != "HS256" || h["typ"] != "JWT" {
		t.Fatalf("unexpected header %s", header)
	}
	mac := hmac.New(sha256.New, []byte(cfg.Secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if !hmac.Equal(sig, mac.Sum(nil)) {
		t.Fatal("signature does not verify with configured secret")
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims struct {
		UserID string `json:"user_id"`
		Issued int64  `json:"iat"`
		Expiry int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-123" || claims.Issued < before || claims.Issued > time.Now().Unix() || claims.Expiry-claims.Issued != 10800 {
		t.Fatalf("unexpected claims %+v", claims)
	}
}

func TestPasswordLoginFailuresNeverIssueToken(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	failure := errors.New("storage offline")
	valid := config.JWTConfig{Secret: "test-secret", Expire: 24}
	for _, tc := range []struct {
		name string
		repo loginRepo
		req  model.PasswordLoginReq
		cfg  config.JWTConfig
		want error
	}{
		{"missing username", loginRepo{}, model.PasswordLoginReq{Password: "123456"}, valid, ErrInvalidLogin},
		{"missing password", loginRepo{}, model.PasswordLoginReq{Username: "cat"}, valid, ErrInvalidLogin},
		{"unknown user", loginRepo{err: data.ErrUserNotFound}, model.PasswordLoginReq{Username: "cat", Password: "123456"}, valid, ErrInvalidCredentials},
		{"wrong password", loginRepo{user: data.UserInfo{Password: string(hash)}}, model.PasswordLoginReq{Username: "cat", Password: "wrong"}, valid, ErrInvalidCredentials},
		{"storage failure", loginRepo{err: failure}, model.PasswordLoginReq{Username: "cat", Password: "123456"}, valid, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NewLoginService(tc.repo, tc.cfg).PasswordLogin(context.Background(), tc.req)
			if !errors.Is(err, tc.want) || result.Token != "" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
	for _, cfg := range []config.JWTConfig{{Expire: 24}, {Secret: "secret"}, {Secret: "secret", Expire: -1}, {Secret: "secret", Expire: int(^uint(0) >> 1)}} {
		result, err := NewLoginService(loginRepo{user: data.UserInfo{UserID: "id", Password: string(hash)}}, cfg).PasswordLogin(context.Background(), model.PasswordLoginReq{Username: "cat", Password: "123456"})
		if err == nil || result.Token != "" {
			t.Fatal("invalid JWT config issued token")
		}
	}
}
