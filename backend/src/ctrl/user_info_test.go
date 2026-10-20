package ctrl

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"testing"
)

type profileService struct {
	err error
	id  string
}

func (s *profileService) GetUserInfo(_ context.Context, id string) (model.UserInfoData, error) {
	s.id = id
	return model.UserInfoData{UserID: id, NickName: "猫", AvatarURL: "avatar", CreateTime: "2024-01-01 12:00:00"}, s.err
}
func TestUserInfoHeaderAndResponse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers []string
		err     error
		code    int
	}{
		{"success", []string{"u1"}, nil, 0}, {"missing", nil, nil, 401}, {"blank", []string{"  "}, nil, 401},
		{"duplicate", []string{"u1", "u2"}, nil, 401}, {"not found", []string{"u1"}, data.ErrUserNotFound, 404},
		{"storage failure", []string{"u1"}, errors.New("private db error"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &profileService{err: tc.err}
			r := gin.New()
			r.GET("/api/v1/user_info/get_user_info", NewUserInfoController(svc).GetUserInfo)
			req := httptest.NewRequest("GET", "/api/v1/user_info/get_user_info?user_id=ignored", nil)
			for _, v := range tc.headers {
				req.Header.Add("user_id", v)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var result struct {
				Code int            `json:"code"`
				Msg  string         `json:"msg"`
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || result.Code != tc.code {
				t.Fatalf("unexpected response %d %s", w.Code, w.Body)
			}
			if tc.code != 0 {
				if result.Data != nil {
					t.Fatal("error leaked data")
				}
				return
			}
			if result.Msg != "success" || svc.id != "u1" || result.Data["user_id"] != "u1" || result.Data["nick_name"] != "猫" {
				t.Fatalf("invalid data %s", w.Body)
			}
			for _, key := range []string{"user_id", "username", "nick_name", "avatar", "vip", "sex", "phone", "email", "introduction", "avatar_url", "user_status", "user_type", "create_time", "update_time"} {
				if _, ok := result.Data[key]; !ok {
					t.Fatalf("missing %s", key)
				}
			}
			if len(result.Data) != 14 {
				t.Fatalf("unexpected fields %s", w.Body)
			}
		})
	}
}
