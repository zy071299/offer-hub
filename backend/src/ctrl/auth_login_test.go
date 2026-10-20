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

type loginService struct{ err error }

func (s loginService) PasswordLogin(context.Context, model.PasswordLoginReq) (model.PasswordLoginData, error) {
	return model.PasswordLoginData{Token: "signed-token", UserInfo: model.PasswordLoginUserInfo{UserID: "id", NickName: "猫", UserStatus: 1, UserType: 2}}, s.err
}
func TestLoginResponseContract(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		err        error
		code       int
		msg        string
	}{
		{"success", `{"username":"cat","password":"123456"}`, nil, 0, "登录成功"},
		{"wrong credentials", `{"username":"cat","password":"wrong"}`, service.ErrInvalidCredentials, 401, "用户名或密码错误"},
		{"invalid", `{"username":"cat","password":"123456"}`, service.ErrInvalidLogin, 400, "用户名或密码不符合要求"},
		{"internal", `{"username":"cat","password":"123456"}`, errors.New("private error"), 500, "服务器内部错误"},
		{"missing", `{}`, nil, 400, "用户名或密码不符合要求"},
		{"malformed", `{`, nil, 400, "用户名或密码不符合要求"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/auth/login", NewLoginController(loginService{err: tc.err}).PasswordLogin)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("POST", "/auth/login", strings.NewReader(tc.body)))
			var response struct {
				Code int                        `json:"code"`
				Msg  string                     `json:"msg"`
				Data map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || response.Code != tc.code || response.Msg != tc.msg {
				t.Fatalf("unexpected response %d %s", w.Code, w.Body)
			}
			if tc.code != 0 {
				if response.Data != nil {
					t.Fatal("failure leaked data")
				}
				return
			}
			if string(response.Data["token"]) != `"signed-token"` || len(response.Data) != 2 {
				t.Fatalf("invalid login data %s", w.Body)
			}
			var info map[string]any
			if err := json.Unmarshal(response.Data["userInfo"], &info); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"userId", "username", "nickName", "avatar", "sex", "vip", "phone", "email", "userStatus", "userType"} {
				if _, ok := info[key]; !ok {
					t.Fatalf("missing %s in %s", key, w.Body)
				}
			}
			if len(info) != 10 || info["userId"] != "id" || info["nickName"] != "猫" {
				t.Fatalf("unexpected user info %s", w.Body)
			}
		})
	}
}
