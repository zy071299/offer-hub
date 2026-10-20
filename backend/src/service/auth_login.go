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

	"golang.org/x/crypto/bcrypt"
	"offer-hub/backend/src/config"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
)

var (
	ErrInvalidLogin       = errors.New("invalid login parameters")
	ErrInvalidCredentials = errors.New("invalid username or password")
)

type LoginRepository interface {
	GetUserByUsername(context.Context, string) (data.UserInfo, error)
}

type LoginService struct {
	repository LoginRepository
	jwtConfig  config.JWTConfig
}

func NewLoginService(repository LoginRepository, jwtConfig config.JWTConfig) *LoginService {
	if repository == nil {
		repository = data.AuthRepository{}
	}
	return &LoginService{repository: repository, jwtConfig: jwtConfig}
}

func (s *LoginService) PasswordLogin(ctx context.Context, req model.PasswordLoginReq) (model.PasswordLoginData, error) {
	if strings.TrimSpace(req.Username) == "" || req.Password == "" {
		return model.PasswordLoginData{}, ErrInvalidLogin
	}
	user, err := s.repository.GetUserByUsername(ctx, req.Username)
	if errors.Is(err, data.ErrUserNotFound) {
		return model.PasswordLoginData{}, ErrInvalidCredentials
	}
	if err != nil {
		return model.PasswordLoginData{}, fmt.Errorf("load login user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return model.PasswordLoginData{}, ErrInvalidCredentials
	}
	token, err := s.signToken(user.UserID)
	if err != nil {
		return model.PasswordLoginData{}, err
	}
	return model.PasswordLoginData{
		Token: token,
		UserInfo: model.PasswordLoginUserInfo{
			UserID: user.UserID, Username: user.Username, NickName: user.NickName,
			Avatar: user.Avatar, Sex: user.Sex, VIP: user.VIP, Phone: user.Phone,
			Email: user.Email, UserStatus: user.UserStatus, UserType: user.UserType,
		},
	}, nil
}

// signToken 按 JWT compact 格式签发 HS256 token，expire 的配置单位为小时。
func (s *LoginService) signToken(userID string) (string, error) {
	const maxExpireHours = int64(1<<63-1) / int64(time.Hour)
	if strings.TrimSpace(s.jwtConfig.Secret) == "" || s.jwtConfig.Expire <= 0 || int64(s.jwtConfig.Expire) > maxExpireHours {
		return "", errors.New("invalid JWT configuration")
	}
	if userID == "" {
		return "", errors.New("login user has no user_id")
	}
	now := time.Now()
	claims := struct {
		UserID    string `json:"user_id"`
		IssuedAt  int64  `json:"iat"`
		ExpiresAt int64  `json:"exp"`
	}{userID, now.Unix(), now.Add(time.Duration(s.jwtConfig.Expire) * time.Hour).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encode JWT claims: %w", err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(s.jwtConfig.Secret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
