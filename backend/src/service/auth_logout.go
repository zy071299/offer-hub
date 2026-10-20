package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"offer-hub/backend/src/config"
	"offer-hub/backend/src/data"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type TokenBlacklist interface {
	IsTokenBlacklisted(context.Context, string) (bool, error)
	BlacklistToken(context.Context, string, time.Duration) (bool, error)
}

type TokenClaims struct {
	UserID    string `json:"user_id"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf"`
}

type LogoutService struct {
	repository TokenBlacklist
	jwtConfig  config.JWTConfig
}

func NewLogoutService(repository TokenBlacklist, jwtConfig config.JWTConfig) *LogoutService {
	if repository == nil {
		repository = data.TokenBlacklistRepository{}
	}
	return &LogoutService{repository: repository, jwtConfig: jwtConfig}
}

// AuthenticateToken 验证签名、有效期及黑名单，可供后续强制鉴权入口复用。
func (s *LogoutService) AuthenticateToken(ctx context.Context, token string) (TokenClaims, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return TokenClaims{}, err
	}
	blocked, err := s.repository.IsTokenBlacklisted(ctx, token)
	if err != nil {
		return TokenClaims{}, fmt.Errorf("authenticate token: %w", err)
	}
	if blocked || claims.ExpiresAt <= time.Now().Unix() {
		return TokenClaims{}, ErrUnauthenticated
	}
	return claims, nil
}

func (s *LogoutService) Logout(ctx context.Context, token string) error {
	claims, err := s.AuthenticateToken(ctx, token)
	if err != nil {
		return err
	}
	ttl := time.Until(time.Unix(claims.ExpiresAt, 0))
	if ttl <= 0 {
		return ErrUnauthenticated
	}
	added, err := s.repository.BlacklistToken(ctx, token, ttl)
	if err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	if !added {
		return ErrUnauthenticated
	}
	return nil
}

func (s *LogoutService) parseToken(token string) (TokenClaims, error) {
	if strings.TrimSpace(s.jwtConfig.Secret) == "" {
		return TokenClaims{}, errors.New("invalid JWT configuration")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenClaims{}, ErrUnauthenticated
	}
	// 严格解码确保 token 编码唯一，避免通过非规范签名编码绕过原文黑名单。
	decode := func(value string) ([]byte, error) {
		decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
		if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != value {
			return nil, ErrUnauthenticated
		}
		return decoded, nil
	}
	header, err := decode(parts[0])
	if err != nil {
		return TokenClaims{}, ErrUnauthenticated
	}
	var metadata struct {
		Algorithm string `json:"alg"`
	}
	if json.Unmarshal(header, &metadata) != nil || metadata.Algorithm != "HS256" {
		return TokenClaims{}, ErrUnauthenticated
	}
	signature, err := decode(parts[2])
	if err != nil {
		return TokenClaims{}, ErrUnauthenticated
	}
	mac := hmac.New(sha256.New, []byte(s.jwtConfig.Secret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return TokenClaims{}, ErrUnauthenticated
	}
	payload, err := decode(parts[1])
	if err != nil {
		return TokenClaims{}, ErrUnauthenticated
	}
	var claims TokenClaims
	if json.Unmarshal(payload, &claims) != nil {
		return TokenClaims{}, ErrUnauthenticated
	}
	now := time.Now().Unix()
	if strings.TrimSpace(claims.UserID) == "" || claims.ExpiresAt <= now || claims.NotBefore > now || claims.IssuedAt > now {
		return TokenClaims{}, ErrUnauthenticated
	}
	return claims, nil
}
