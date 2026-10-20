package comment

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"offer-hub/backend/src/model"
	"strings"
	"testing"
)

type commentServiceStub struct {
	req   model.ListCommentsReq
	calls int
	err   error
}

func (s *commentServiceStub) ListComments(_ context.Context, req model.ListCommentsReq) (model.ListCommentsData, error) {
	s.req = req
	s.calls++
	return model.ListCommentsData{List: []model.CommentInfo{}}, s.err
}
func TestListCommentsBindingAndErrors(t *testing.T) {
	for _, tc := range []struct {
		query string
		code  int
	}{
		{"target_type=1&target_id=q1", 0}, {"target_type=2&target_id=x&parent_id=c1&sort_by=thumbs_up&sort_order=asc&page=2&page_size=3&sub_comment_page=2&sub_comment_size=1", 0},
		{"", 400}, {"target_type=0&target_id=q1", 400}, {"target_type=3&target_id=q1", 400}, {"target_type=x&target_id=q1", 400}, {"target_type=1&target_id=%20", 400},
		{"target_type=1&target_id=q1&page=0", 400}, {"target_type=1&target_id=q1&page_size=-1", 400}, {"target_type=1&target_id=q1&sub_comment_page=0", 400}, {"target_type=1&target_id=q1&sub_comment_size=0", 400},
		{"target_type=1&target_id=q1&sort_by=view_count", 400}, {"target_type=1&target_id=q1&sort_order=up", 400}, {"target_type=1&target_id=q1&page=9223372036854775807&page_size=20", 400}, {"target_type=1&target_id=bad;query", 400},
	} {
		t.Run(tc.query, func(t *testing.T) {
			s := &commentServiceStub{}
			r := gin.New()
			r.GET("/comments", NewController(s).ListComments)
			req := httptest.NewRequest("GET", "/comments?"+tc.query+"&user_id=forged", nil)
			req.Header.Set("user_id", "viewer")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var body struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || body.Code != tc.code {
				t.Fatalf("unexpected response %d %s", w.Code, w.Body)
			}
			if tc.code == 0 {
				if s.req.UserID != "viewer" || s.req.Page <= 0 || s.req.SubCommentSize <= 0 {
					t.Fatalf("incorrect defaults %+v", s.req)
				}
			} else if s.calls != 0 {
				t.Fatal("invalid query reached service")
			}
		})
	}
	s := &commentServiceStub{err: errors.New("private failure")}
	r := gin.New()
	r.GET("/comments", NewController(s).ListComments)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/comments?target_type=1&target_id=q1", nil))
	if !strings.Contains(w.Body.String(), `"code":500`) || !strings.Contains(w.Body.String(), `"data":null`) || strings.Contains(w.Body.String(), "private") {
		t.Fatalf("incorrect failure %s", w.Body)
	}
}
