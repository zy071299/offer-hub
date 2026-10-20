package comment

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

type commentUpdaterStub struct {
	err   error
	user  string
	req   model.UpdateCommentReq
	calls int
}

func (s *commentUpdaterStub) UpdateComment(_ context.Context, user string, req model.UpdateCommentReq) (model.UpdateCommentData, error) {
	s.calls++
	s.user = user
	s.req = req
	return model.UpdateCommentData{CommentID: req.CommentID}, s.err
}
func TestUpdateCommentHandler(t *testing.T) {
	valid := `{"comment_id":"c1","content":"new content","user_id":"forged"}`
	for _, tc := range []struct {
		name, body, user string
		err              error
		code             int
	}{
		{"success", valid, "u1", nil, 0}, {"no identity", valid, "", nil, 401},
		{"malformed", "{", "u1", nil, 400}, {"missing id", `{"content":"x"}`, "u1", nil, 400},
		{"blank content", `{"comment_id":"c1","content":" "}`, "u1", nil, 400},
		{"other author", valid, "u1", service.ErrCommentForbidden, 403}, {"missing", valid, "u1", service.ErrCommentNotFound, 404},
		{"invalid", valid, "u1", service.ErrInvalidCommentInput, 400}, {"failed", valid, "u1", errors.New("private error"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &commentUpdaterStub{err: tc.err}
			r := gin.New()
			r.POST("/update", NewUpdateController(s).UpdateComment)
			req := httptest.NewRequest("POST", "/update", strings.NewReader(tc.body))
			if tc.user != "" {
				req.Header.Set("user_id", tc.user)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var result struct {
				Code int                      `json:"code"`
				Msg  string                   `json:"msg"`
				Data *model.UpdateCommentData `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || result.Code != tc.code {
				t.Fatalf("unexpected response %d %s", w.Code, w.Body)
			}
			if tc.code == 0 {
				if result.Msg != "success" || result.Data == nil || result.Data.CommentID != "c1" || s.user != "u1" || s.req.Content != "new content" {
					t.Fatalf("incorrect response %s", w.Body)
				}
			} else if result.Data != nil || strings.Contains(w.Body.String(), "private") {
				t.Fatalf("error leaked %s", w.Body)
			}
			if (tc.user == "" || tc.name == "malformed" || tc.name == "missing id" || tc.name == "blank content") && s.calls != 0 {
				t.Fatal("invalid request reached service")
			}
		})
	}
}
