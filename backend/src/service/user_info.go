package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
)

type UserInfoRepository interface {
	GetUserByID(context.Context, string) (data.UserInfo, error)
}

var ErrUserInfoNotFound = data.ErrUserNotFound

type UserInfoService struct{ repository UserInfoRepository }

func NewUserInfoService(repository UserInfoRepository) *UserInfoService {
	if repository == nil {
		repository = data.AuthRepository{}
	}
	return &UserInfoService{repository: repository}
}

func (s *UserInfoService) GetUserInfo(ctx context.Context, userID string) (model.UserInfoData, error) {
	if strings.TrimSpace(userID) == "" {
		return model.UserInfoData{}, ErrUnauthenticated
	}
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return model.UserInfoData{}, fmt.Errorf("get user info: %w", err)
	}
	return model.UserInfoData{
		UserID: user.UserID, Username: user.Username, NickName: user.NickName,
		Avatar: user.Avatar, VIP: user.VIP, Sex: user.Sex, Phone: user.Phone,
		Email: user.Email, Introduction: user.Introduction, AvatarURL: user.Avatar,
		UserStatus: user.UserStatus, UserType: user.UserType,
		CreateTime: formatUserInfoTime(user.CreateTime), UpdateTime: formatUserInfoTime(user.UpdateTime),
	}, nil
}

func formatUserInfoTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}
