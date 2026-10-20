package ctrl

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"offer-hub/backend/src/model"
	"offer-hub/backend/src/service"
	"strings"
	"testing"
)

type registerService struct {
	err   error
	calls int
}

func (s *registerService) PasswordRegister(context.Context, model.PasswordRegisterReq) error {
	s.calls++
	return s.err
}

func TestRegisterResponses(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		err        error
		code       int
		msg        string
		calls      int
	}{
		{"success", `{"username":"cat","password":"123456"}`, nil, 0, "注册成功", 1},
		{"duplicate", `{"username":"cat","password":"123456"}`, service.ErrUsernameExists, 500, "用户名已存在", 1},
		{"invalid", `{"username":"cat","password":"123456"}`, service.ErrInvalidRegister, 400, "用户名或密码不符合要求", 1},
		{"internal", `{"username":"cat","password":"123456"}`, errors.New("private storage details"), 500, "服务器内部错误", 1},
		{"missing", `{"username":"cat"}`, nil, 400, "用户名或密码不符合要求", 0},
		{"malformed", `{`, nil, 400, "用户名或密码不符合要求", 0},
		{"wrong type", `{"username":123,"password":"123456"}`, nil, 400, "用户名或密码不符合要求", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &registerService{err: tc.err}
			router := gin.New()
			router.POST("/auth/register", NewAuthController(s).PasswordRegister)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/auth/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
			var result struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
				Data any    `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || result.Code != tc.code || result.Msg != tc.msg || result.Data != nil || s.calls != tc.calls {
				t.Fatalf("unexpected response %d %s calls=%d", w.Code, w.Body, s.calls)
			}
		})
	}
}
