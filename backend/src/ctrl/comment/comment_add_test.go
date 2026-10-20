package comment

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"offer-hub/backend/src/service"
	"strings"
	"testing"
)

type commentAdderStub struct {
	userID string
	req    model.AddCommentReq
	err    error
	calls  int
}

func (s *commentAdderStub) AddComment(_ context.Context, id string, req model.AddCommentReq) (model.AddCommentData, error) {
	s.calls++
	s.userID = id
	s.req = req
	return model.AddCommentData{CommentID: "new", Comment: model.CommentInfo{CommentID: "new", UserName: "cat", SubComments: []model.CommentInfo{}}}, s.err
}
func TestAddCommentHandler(t *testing.T) {
	valid := `{"target_type":1,"target_id":"q1","content":"hello","user_id":"forged"}`
	for _, tc := range []struct {
		name, body, userID string
		err                error
		code               int
	}{
		{"success", valid, "u1", nil, 0}, {"missing identity", valid, "", nil, 401},
		{"invalid JSON", "{", "u1", nil, 400}, {"invalid input", valid, "u1", service.ErrInvalidCommentInput, 400},
		{"missing parent", valid, "u1", data.ErrCommentParentNotFound, 404}, {"failure", valid, "u1", errors.New("private failure"), 500},
		{"compensation failure", valid, "u1", errors.Join(service.ErrCommentParentNotFound, service.ErrCommentCompensationFailed), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &commentAdderStub{err: tc.err}
			r := gin.New()
			r.POST("/add", NewAddController(s).AddComment)
			req := httptest.NewRequest("POST", "/add", strings.NewReader(tc.body))
			if tc.userID != "" {
				req.Header.Set("user_id", tc.userID)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var result struct {
				Code int                   `json:"code"`
				Msg  string                `json:"msg"`
				Data *model.AddCommentData `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || result.Code != tc.code {
				t.Fatalf("unexpected response %d %s", w.Code, w.Body)
			}
			if tc.code == 0 {
				if s.userID != "u1" || result.Msg != "success" || result.Data == nil || result.Data.CommentID != "new" || result.Data.Comment.UserName != "cat" {
					t.Fatalf("incorrect response %s", w.Body)
				}
			} else if result.Data != nil || strings.Contains(w.Body.String(), "private") {
				t.Fatalf("leaked failure %s", w.Body)
			}
			if tc.userID == "" && s.calls != 0 {
				t.Fatal("unauthenticated request reached service")
			}
		})
	}
}
