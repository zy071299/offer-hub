package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
)

var (
	ErrInvalidRegister = errors.New("invalid registration parameters")
	ErrUsernameExists  = data.ErrUsernameExists
)

type AuthRepository interface {
	UsernameExists(context.Context, string) (bool, error)
	CreateUser(context.Context, *data.UserInfo) error
}

type AuthService struct{ repository AuthRepository }

func NewAuthService(repository AuthRepository) *AuthService {
	if repository == nil {
		repository = data.AuthRepository{}
	}
	return &AuthService{repository: repository}
}

func (s *AuthService) PasswordRegister(ctx context.Context, req model.PasswordRegisterReq) error {
	// 按字符数校验；bcrypt 本身最多接受 72 字节，超出作为参数错误返回。
	if !utf8.ValidString(req.Username) || !utf8.ValidString(req.Password) || strings.TrimSpace(req.Username) == "" || utf8.RuneCountInString(req.Username) > 50 || utf8.RuneCountInString(req.Password) < 6 || len(req.Password) > 72 {
		return ErrInvalidRegister
	}
	exists, err := s.repository.UsernameExists(ctx, req.Username)
	if err != nil {
		return fmt.Errorf("check registration username: %w", err)
	}
	if exists {
		return ErrUsernameExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return fmt.Errorf("hash registration password: %w", err)
	}
	// RFC 4122 UUID v4，使用密码学随机源。
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return fmt.Errorf("generate user UUID: %w", err)
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	user := &data.UserInfo{
		UserID:   fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]),
		Username: req.Username, Password: string(hash), NickName: req.Username,
	}
	if err := s.repository.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("register user: %w", err)
	}
	return nil
}
