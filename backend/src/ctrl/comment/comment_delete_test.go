package comment

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"offer-hub/backend/src/service"
	"strings"
	"testing"
)

type commentDeleterStub struct {
	err      error
	user, id string
	calls    int
}

func (s *commentDeleterStub) DeleteComment(_ context.Context, user, id string) error {
	s.calls++
	s.user = user
	s.id = id
	return s.err
}
func TestDeleteCommentHandler(t *testing.T) {
	for _, tc := range []struct {
		name, body, user string
		err              error
		code             int
	}{
		{"success", `{"comment_id":"c1","user_id":"forged"}`, "u1", nil, 0},
		{"no identity", `{"comment_id":"c1"}`, "", nil, 401},
		{"malformed", "{", "u1", nil, 400}, {"missing id", `{}`, "u1", nil, 400}, {"blank id", `{"comment_id":" "}`, "u1", nil, 400},
		{"forbidden", `{"comment_id":"c1"}`, "u1", service.ErrCommentForbidden, 403},
		{"missing", `{"comment_id":"c1"}`, "u1", service.ErrCommentNotFound, 404},
		{"failed", `{"comment_id":"c1"}`, "u1", errors.New("private failure"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &commentDeleterStub{err: tc.err}
			r := gin.New()
			r.POST("/delete", NewDeleteController(s).DeleteComment)
			req := httptest.NewRequest("POST", "/delete", strings.NewReader(tc.body))
			if tc.user != "" {
				req.Header.Set("user_id", tc.user)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var result map[string]json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			var code int
			json.Unmarshal(result["code"], &code)
			if w.Code != 200 || code != tc.code {
				t.Fatalf("unexpected response %d %s", w.Code, w.Body)
			}
			if tc.code == 0 {
				if len(result) != 2 || string(result["msg"]) != `"success"` || s.user != "u1" || s.id != "c1" {
					t.Fatalf("incorrect success %s", w.Body)
				}
			} else if string(result["data"]) != "null" || strings.Contains(w.Body.String(), "private") {
				t.Fatalf("incorrect error %s", w.Body)
			}
			if (tc.code == 400 || tc.user == "") && s.calls != 0 {
				t.Fatal("invalid request reached service")
			}
		})
	}
}
