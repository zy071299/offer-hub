package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
)

type registerRepo struct {
	exists               bool
	lookupErr, insertErr error
	inserted             *data.UserInfo
	queries              int
}

func (r *registerRepo) UsernameExists(context.Context, string) (bool, error) {
	r.queries++
	return r.exists, r.lookupErr
}
func (r *registerRepo) CreateUser(_ context.Context, user *data.UserInfo) error {
	r.inserted = user
	return r.insertErr
}

func TestPasswordRegisterCreatesHashedUser(t *testing.T) {
	r := &registerRepo{}
	req := model.PasswordRegisterReq{Username: strings.Repeat("猫", 50), Password: "123456"}
	if err := NewAuthService(r).PasswordRegister(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	u := r.inserted
	if u == nil || u.Username != req.Username || u.NickName != req.Username {
		t.Fatalf("unexpected user: %+v", u)
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(u.UserID) {
		t.Fatalf("invalid UUID: %s", u.UserID)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		t.Fatal(err)
	}
	if cost, err := bcrypt.Cost([]byte(u.Password)); err != nil || cost != 12 {
		t.Fatalf("cost=%d err=%v", cost, err)
	}
}

func TestPasswordRegisterRejectsInvalidInputBeforeStorage(t *testing.T) {
	for _, req := range []model.PasswordRegisterReq{
		{Username: "", Password: "123456"}, {Username: "   ", Password: "123456"},
		{Username: strings.Repeat("猫", 51), Password: "123456"}, {Username: "cat", Password: "12345"},
		{Username: "cat", Password: strings.Repeat("a", 73)},
	} {
		r := &registerRepo{}
		if err := NewAuthService(r).PasswordRegister(context.Background(), req); !errors.Is(err, ErrInvalidRegister) {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.queries != 0 || r.inserted != nil {
			t.Fatal("invalid input accessed storage")
		}
	}
}

func TestPasswordRegisterStorageErrors(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, tc := range []struct {
		name string
		repo registerRepo
		want error
	}{
		{"existing", registerRepo{exists: true}, ErrUsernameExists},
		{"lookup failure", registerRepo{lookupErr: failure}, failure},
		{"concurrent duplicate", registerRepo{insertErr: data.ErrUsernameExists}, ErrUsernameExists},
		{"insert failure", registerRepo{insertErr: failure}, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := NewAuthService(&tc.repo).PasswordRegister(context.Background(), model.PasswordRegisterReq{Username: "cat", Password: "123456"}); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if (tc.repo.exists || tc.repo.lookupErr != nil) && tc.repo.inserted != nil {
				t.Fatal("inserted after failed lookup")
			}
		})
	}
}
