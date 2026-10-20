package router

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateCommentRouteRequiresJWT(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/comment/update", strings.NewReader(`{"comment_id":"c1","content":"changed"}`))
	req.Header.Set("user_id", "forged-user")
	w := httptest.NewRecorder()
	New().ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":401`) || req.Header.Get("user_id") != "" {
		t.Fatalf("unverified identity accepted: %d %s", w.Code, w.Body)
	}
}

func TestDeleteCommentRouteRequiresJWT(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/comment/delete", strings.NewReader(`{"comment_id":"c1"}`))
	req.Header.Set("user_id", "forged-user")
	w := httptest.NewRecorder()
	New().ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":401`) || req.Header.Get("user_id") != "" {
		t.Fatalf("unverified identity accepted: %d %s", w.Code, w.Body)
	}
}

func TestAddCommentRouteRequiresJWT(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/comment/add", strings.NewReader(`{"target_type":1,"target_id":"q1","content":"hello"}`))
	req.Header.Set("user_id", "forged-user")
	w := httptest.NewRecorder()
	New().ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":401`) || req.Header.Get("user_id") != "" {
		t.Fatalf("unverified identity accepted: %d %s", w.Code, w.Body)
	}
}

func TestCommentsRouteAllowsGuests(t *testing.T) {
	for _, authorization := range []string{"", "Bearer invalid-token"} {
		req := httptest.NewRequest("GET", "/api/v1/open/list_comments?target_type=1&target_id=q1", nil)
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		w := httptest.NewRecorder()
		New().ServeHTTP(w, req)
		// 无数据库时业务返回 500；游客及无效 token 均应到达 Handler。
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":500`) {
			t.Fatalf("incorrect route authentication: %d %s", w.Code, w.Body)
		}
	}
}
