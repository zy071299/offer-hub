package router

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuestionGroupUsesSoftAuthentication(t *testing.T) {
	for _, path := range []string{
		"/api/v1/question/all/list", "/api/v1/question/list",
		"/api/v1/question/meta/list", "/api/v1/question/detail?question_id=q1",
		"/api/v1/question/hot/list",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			req.Header.Set("Authorization", "Bearer invalid-token")
			req.Header.Set("user_id", "forged-user")
			w := httptest.NewRecorder()
			New().ServeHTTP(w, req)
			// 无数据库时应到达 Handler 返回 500，且伪造身份已被软鉴权清除。
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":500`) || req.Header.Get("user_id") != "" {
				t.Fatalf("unexpected soft authentication: %d %s identity=%q", w.Code, w.Body, req.Header.Get("user_id"))
			}
		})
	}
}
