package service

import (
	"context"
	"errors"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"testing"
	"time"
)

type profileRepository struct {
	user data.UserInfo
	err  error
	id   string
}

func (r *profileRepository) GetUserByID(_ context.Context, id string) (data.UserInfo, error) {
	r.id = id
	return r.user, r.err
}
func TestGetUserInfoMapsAllFields(t *testing.T) {
	r := &profileRepository{user: data.UserInfo{UserID: "u1", Username: "cat", Password: "private-hash", NickName: "猫", Avatar: "avatar", VIP: true, Sex: 2, Phone: "phone", Email: "email", Introduction: "intro", UserStatus: 1, UserType: 2, CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), UpdateTime: time.Date(2024, 1, 2, 13, 0, 0, 0, time.UTC)}}
	got, err := NewUserInfoService(r).GetUserInfo(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	want := model.UserInfoData{UserID: "u1", Username: "cat", NickName: "猫", Avatar: "avatar", AvatarURL: "avatar", VIP: true, Sex: 2, Phone: "phone", Email: "email", Introduction: "intro", UserStatus: 1, UserType: 2, CreateTime: "2024-01-01 12:00:00", UpdateTime: "2024-01-02 13:00:00"}
	if got != want || r.id != "u1" {
		t.Fatalf("got %+v want %+v", got, want)
	}
}
func TestGetUserInfoErrorsAndZeroTimes(t *testing.T) {
	r := &profileRepository{}
	if _, err := NewUserInfoService(r).GetUserInfo(context.Background(), "  "); !errors.Is(err, ErrUnauthenticated) || r.id != "" {
		t.Fatalf("empty identity accepted: %v", err)
	}
	for _, failure := range []error{data.ErrUserNotFound, errors.New("storage failure")} {
		r.err = failure
		if _, err := NewUserInfoService(r).GetUserInfo(context.Background(), "u1"); !errors.Is(err, failure) {
			t.Fatalf("error lost: %v", err)
		}
	}
	r.err = nil
	result, err := NewUserInfoService(r).GetUserInfo(context.Background(), "u1")
	if err != nil || result.CreateTime != "" || result.UpdateTime != "" {
		t.Fatalf("zero times: %+v %v", result, err)
	}
}
