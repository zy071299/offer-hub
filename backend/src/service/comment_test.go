package service

import (
	"context"
	"errors"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"reflect"
	"testing"
	"time"
)

type commentRepoStub struct {
	comments, children     []data.Comment
	total                  int64
	filter                 data.CommentFilter
	parentIDs, likedIDs    []string
	likedUser              string
	likes                  map[string]bool
	err, childErr, likeErr error
	childCalls             int
}

func (r *commentRepoStub) ListComments(_ context.Context, f data.CommentFilter) ([]data.Comment, int64, error) {
	r.filter = f
	return r.comments, r.total, r.err
}
func (r *commentRepoStub) ListCommentChildren(_ context.Context, f data.CommentFilter, ids []string) ([]data.Comment, error) {
	r.parentIDs = ids
	r.childCalls++
	return r.children, r.childErr
}
func (r *commentRepoStub) GetLikedComments(_ context.Context, user string, ids []string) (map[string]bool, error) {
	r.likedUser = user
	r.likedIDs = ids
	return r.likes, r.likeErr
}

type commentUsersStub struct {
	profiles map[string]data.CommentUser
	ids      []string
	err      error
}

func (r *commentUsersStub) GetCommentUsers(_ context.Context, ids []string) (map[string]data.CommentUser, error) {
	r.ids = ids
	return r.profiles, r.err
}
func commentReq() model.ListCommentsReq {
	return model.ListCommentsReq{TargetType: 1, TargetID: "q1", Page: 1, PageSize: 20, SubCommentPage: 1, SubCommentSize: 2, SortBy: "create_time", SortOrder: "desc"}
}
func TestListCommentsBuildsNestedPagesAndBatchesEnrichment(t *testing.T) {
	r := &commentRepoStub{comments: []data.Comment{{CommentID: "c1", UserID: "u1", Content: "top", Status: 2, CreateTime: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}, {CommentID: "c2", UserID: "missing"}}, total: 50,
		children: []data.Comment{{CommentID: "s1", ParentID: "c1", UserID: "u2", ReplyTo: "u1"}, {CommentID: "s2", ParentID: "c1", UserID: "u2"}, {CommentID: "s3", ParentID: "c1", UserID: "u3"}, {CommentID: "t1", ParentID: "c2", UserID: "u2"}}, likes: map[string]bool{"c1": true, "s1": true}}
	users := &commentUsersStub{profiles: map[string]data.CommentUser{"u1": {Username: "cat", Avatar: "avatar"}, "u2": {Username: "reply"}}}
	req := commentReq()
	req.UserID = "viewer"
	got, err := NewCommentService(r, users).ListComments(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 50 || len(got.List) != 2 || got.List[0].SubCommentTotal != 3 || len(got.List[0].SubComments) != 2 || got.List[1].SubCommentTotal != 1 {
		t.Fatalf("incorrect nested pages %+v", got)
	}
	top := got.List[0]
	sub := top.SubComments[0]
	if top.UserName != "cat" || top.UserAvatar != "avatar" || !top.UserLiked || top.CreateTime != "2024-01-01 12:00:00" || sub.ReplyToName != "cat" || !sub.UserLiked || sub.SubComments == nil || len(sub.SubComments) != 0 {
		t.Fatalf("incorrect enrichment %+v %+v", top, sub)
	}
	if got.List[1].UserName != "" || !reflect.DeepEqual(r.parentIDs, []string{"c1", "c2"}) || r.likedUser != "viewer" || len(r.likedIDs) != 5 || len(users.ids) != 3 {
		t.Fatalf("invalid batching ids=%v likes=%v", users.ids, r.likedIDs)
	}
	req.SubCommentPage = 2
	req.UserID = ""
	r.likedUser = ""
	r.likedIDs = nil
	got, err = NewCommentService(r, users).ListComments(context.Background(), req)
	if err != nil || len(got.List[0].SubComments) != 1 || got.List[0].SubComments[0].CommentID != "s3" || got.List[1].SubComments == nil || len(got.List[1].SubComments) != 0 || r.likedIDs != nil {
		t.Fatalf("incorrect second nested page %+v %v", got, err)
	}
}
func TestListCommentsChildModeAndEmptyList(t *testing.T) {
	r := &commentRepoStub{comments: []data.Comment{{CommentID: "s1", ParentID: "c1", UserID: "u1"}}, total: 9}
	u := &commentUsersStub{}
	req := commentReq()
	req.ParentID = "c1"
	req.Page = 2
	got, err := NewCommentService(r, u).ListComments(context.Background(), req)
	if err != nil || got.Total != 9 || r.filter.ParentID != "c1" || r.filter.Page != 2 || r.childCalls != 0 || got.List[0].SubComments == nil || len(got.List[0].SubComments) != 0 || got.List[0].SubCommentTotal != 0 {
		t.Fatalf("incorrect child mode %+v %v", got, err)
	}
	r.comments = nil
	r.total = 0
	u.ids = nil
	got, err = NewCommentService(r, u).ListComments(context.Background(), commentReq())
	if err != nil || got.List == nil || len(got.List) != 0 || len(u.ids) != 0 || r.childCalls != 0 {
		t.Fatalf("incorrect empty list %+v %v", got, err)
	}
}
func TestListCommentsErrors(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, stage := range []string{"list", "children", "users", "likes"} {
		t.Run(stage, func(t *testing.T) {
			r := &commentRepoStub{comments: []data.Comment{{CommentID: "c1", UserID: "u1"}}}
			u := &commentUsersStub{}
			switch stage {
			case "list":
				r.err = failure
			case "children":
				r.childErr = failure
			case "users":
				u.err = failure
			case "likes":
				r.likeErr = failure
			}
			req := commentReq()
			req.UserID = "viewer"
			if _, err := NewCommentService(r, u).ListComments(context.Background(), req); !errors.Is(err, failure) {
				t.Fatalf("error lost: %v", err)
			}
		})
	}
	req := commentReq()
	req.SortBy = "view_count"
	if _, err := NewCommentService(&commentRepoStub{}, &commentUsersStub{}).ListComments(context.Background(), req); !errors.Is(err, ErrInvalidCommentQuery) {
		t.Fatalf("invalid sort accepted %v", err)
	}
}
