package router

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestAllQuestionListRouteAllowsGuests(t *testing.T) {
	// 不建立数据库连接，确认请求已到达控制器并采用统一错误响应。
	router := New()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/question/all/list?job_name=backend", nil))
	var response struct {
		Code int `json:"code"`
		Data any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || response.Code != 500 || response.Data != nil {
		t.Fatalf("unexpected route response: %d %s", w.Code, w.Body)
	}
}

func TestQuestionListRouteAllowsGuests(t *testing.T) {
	router := New()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/question/list?page=1&page_size=20", nil))
	var response struct {
		Code int `json:"code"`
		Data any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	// 无数据库时应到达控制器并返回业务错误，而不是路由 404 或鉴权 401。
	if w.Code != 200 || response.Code != 500 || response.Data != nil {
		t.Fatalf("unexpected list route response: %d %s", w.Code, w.Body)
	}
}

func TestQuestionMetaListRouteAllowsGuests(t *testing.T) {
	router := New()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/question/meta/list?page=1&page_size=20", nil))
	var response struct {
		Code int `json:"code"`
		Data any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	// 未初始化数据库时，已注册的接口应返回统一业务错误。
	if w.Code != 200 || response.Code != 500 || response.Data != nil {
		t.Fatalf("unexpected meta route response: %d %s", w.Code, w.Body)
	}
}

func TestQuestionDetailRoute(t *testing.T) {
	router := New()
	for _, tc := range []struct {
		path string
		code int
	}{
		{"/api/v1/question/detail", 400},
		{"/api/v1/question/detail?question_id=q1", 500},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		var response struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || response.Code != tc.code {
			t.Fatalf("unexpected detail route response: %d %s", w.Code, w.Body)
		}
	}
}

func TestHotQuestionsRoute(t *testing.T) {
	router := New()
	for _, tc := range []struct {
		path string
		code int
	}{
		{"/api/v1/question/hot/list?limit=0", 400},
		{"/api/v1/question/hot/list", 500},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || body.Code != tc.code {
			t.Fatalf("unexpected hot route response: %d %s", w.Code, w.Body)
		}
	}
}
