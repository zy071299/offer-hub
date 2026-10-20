package service

import (
	"context"
	"errors"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"regexp"
	"strings"
	"testing"
)

type commentCreatorStub struct {
	parent               data.Comment
	parentErr, createErr error
	created              *data.Comment
	parentID, targetID   string
	targetType           int
	parentCalls          int
}

func (r *commentCreatorStub) GetCommentParent(_ context.Context, typ int, targetID, parentID string) (data.Comment, error) {
	r.parentCalls++
	r.targetType = typ
	r.targetID = targetID
	r.parentID = parentID
	return r.parent, r.parentErr
}
func (r *commentCreatorStub) CreateComment(_ context.Context, c data.Comment) error {
	r.created = &c
	return r.createErr
}
func addUsers() *commentUsersStub {
	return &commentUsersStub{profiles: map[string]data.CommentUser{"u1": {UserID: "u1", Username: "cat", Avatar: "avatar"}, "u2": {UserID: "u2", Username: "reply"}}}
}
func TestAddCommentFiltersAndReturnsStoredComment(t *testing.T) {
	for _, parentID := range []string{"", "root"} {
		r := &commentCreatorStub{parent: data.Comment{CommentID: "root", TargetType: 1, TargetID: "q1", Status: 2}}
		req := model.AddCommentReq{TargetType: 1, TargetID: "q1", Content: "bad🙂", ParentID: parentID}
		if parentID != "" {
			req.ReplyTo = "u2"
		}
		filtered := false
		filter := func(s string) string { filtered = true; return strings.ReplaceAll(s, "bad", "***") }
		got, err := NewAddCommentService(r, addUsers(), filter).AddComment(context.Background(), "u1", req)
		if err != nil {
			t.Fatal(err)
		}
		c := r.created
		if c == nil || !filtered || c.Content != "***🙂" || c.Status != 2 || c.UserID != "u1" || c.QuestionID != "q1" || c.ParentID != parentID || c.ChildCount != 0 || c.ThumbsUp != 0 || c.ViewCount != 0 || c.CreateTime.IsZero() || !c.CreateTime.Equal(c.UpdateTime) {
			t.Fatalf("incorrect stored comment %+v", c)
		}
		if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(c.CommentID) || c.ID != c.CommentID {
			t.Fatalf("invalid IDs %+v", c)
		}
		if got.CommentID != c.CommentID || got.Comment.CommentID != c.CommentID || got.Comment.Content != c.Content || got.Comment.UserName != "cat" || got.Comment.UserAvatar != "avatar" || got.Comment.SubComments == nil || got.Comment.SubCommentTotal != 0 || got.Comment.UserLiked || got.Comment.CreateTime == "" {
			t.Fatalf("incorrect response %+v", got)
		}
		if parentID != "" && (got.Comment.ReplyToName != "reply" || r.parentID != "root" || r.targetType != 1 || r.targetID != "q1") {
			t.Fatalf("incorrect reply %+v", got)
		}
		if parentID == "" && r.parentCalls != 0 {
			t.Fatal("top comment queried parent")
		}
	}
}
func TestAddCommentRejectsInvalidInputAndStorageFailures(t *testing.T) {
	valid := model.AddCommentReq{TargetType: 1, TargetID: "q1", Content: "hello"}
	for _, req := range []model.AddCommentReq{{TargetType: 0, TargetID: "q1", Content: "x"}, {TargetType: 3, TargetID: "q1", Content: "x"}, {TargetType: 1, Content: "x"}, {TargetType: 1, TargetID: "q1", Content: "  "}, {TargetType: 1, TargetID: "q1", Content: "x", ReplyTo: "u2"}} {
		r := &commentCreatorStub{}
		if _, err := NewAddCommentService(r, addUsers(), func(s string) string { return s }).AddComment(context.Background(), "u1", req); !errors.Is(err, ErrInvalidCommentInput) || r.created != nil {
			t.Fatalf("invalid input accepted %v", err)
		}
	}
	failure := errors.New("database failed")
	for _, stage := range []string{"missing identity", "missing author", "users", "parent", "insert", "empty filtered content"} {
		t.Run(stage, func(t *testing.T) {
			r := &commentCreatorStub{}
			users := addUsers()
			id := "u1"
			req := valid
			want := failure
			filter := func(s string) string { return s }
			switch stage {
			case "missing identity":
				id = ""
				want = ErrUnauthenticated
			case "missing author":
				users.profiles = nil
				want = ErrUnauthenticated
			case "users":
				users.err = failure
			case "parent":
				req.ParentID = "missing"
				r.parentErr = data.ErrCommentParentNotFound
				want = data.ErrCommentParentNotFound
			case "insert":
				r.createErr = failure
			case "empty filtered content":
				filter = func(string) string { return "" }
				want = ErrInvalidCommentInput
			}
			if _, err := NewAddCommentService(r, users, filter).AddComment(context.Background(), id, req); !errors.Is(err, want) {
				t.Fatalf("got %v want %v", err, want)
			}
			if stage != "insert" && r.created != nil {
				t.Fatal("failed validation wrote comment")
			}
		})
	}
}
