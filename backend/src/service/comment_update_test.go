package service

import (
	"context"
	"errors"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"strings"
	"testing"
)

type commentUpdateRepo struct {
	comment                   data.Comment
	readErr, writeErr         error
	readID, id, user, content string
	writes                    int
}

func (r *commentUpdateRepo) GetCommentByID(_ context.Context, id string) (data.Comment, error) {
	r.readID = id
	return r.comment, r.readErr
}
func (r *commentUpdateRepo) UpdateCommentContent(_ context.Context, id, user, content string) error {
	r.writes++
	r.id = id
	r.user = user
	r.content = content
	return r.writeErr
}
func TestUpdateCommentFiltersContentAndPreservesOwnership(t *testing.T) {
	r := &commentUpdateRepo{comment: data.Comment{CommentID: "c1", UserID: "u1", Status: 2}}
	result, err := NewUpdateCommentService(r, func(s string) string { return strings.ReplaceAll(s, "bad", "***") }).UpdateComment(context.Background(), "u1", model.UpdateCommentReq{CommentID: "c1", Content: "bad🙂"})
	if err != nil || result.CommentID != "c1" || r.readID != "c1" || r.id != "c1" || r.user != "u1" || r.content != "***🙂" || r.writes != 1 {
		t.Fatalf("incorrect update %+v %+v %v", result, r, err)
	}
}
func TestUpdateCommentRejectsInvalidOrUnauthorizedRequests(t *testing.T) {
	for _, tc := range []struct {
		name, user, id, content string
		status                  int
		want                    error
	}{
		{"missing user", "", "c1", "hello", 2, ErrUnauthenticated},
		{"missing id", "u1", " ", "hello", 2, ErrInvalidCommentInput},
		{"empty content", "u1", "c1", "  ", 2, ErrInvalidCommentInput},
		{"other author", "u2", "c1", "hello", 2, ErrCommentForbidden},
		{"deleted", "u1", "c1", "hello", 5, ErrCommentNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &commentUpdateRepo{comment: data.Comment{UserID: "u1", Status: tc.status}}
			filtered := false
			_, err := NewUpdateCommentService(r, func(s string) string { filtered = true; return s }).UpdateComment(context.Background(), tc.user, model.UpdateCommentReq{CommentID: tc.id, Content: tc.content})
			if !errors.Is(err, tc.want) || r.writes != 0 || filtered {
				t.Fatalf("invalid request processed: %v writes=%d filter=%v", err, r.writes, filtered)
			}
		})
	}
}
func TestUpdateCommentStorageAndFilterFailures(t *testing.T) {
	failure := errors.New("storage failed")
	for _, tc := range []struct {
		name              string
		readErr, writeErr error
		filter            func(string) string
		want              error
	}{
		{"not found", data.ErrCommentNotFound, nil, func(s string) string { return s }, data.ErrCommentNotFound},
		{"read failure", failure, nil, func(s string) string { return s }, failure},
		{"write failure", nil, failure, func(s string) string { return s }, failure},
		{"concurrent deletion", nil, data.ErrCommentNotFound, func(s string) string { return s }, data.ErrCommentNotFound},
		{"empty filtered content", nil, nil, func(string) string { return "" }, ErrInvalidCommentInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &commentUpdateRepo{comment: data.Comment{UserID: "u1", Status: 2}, readErr: tc.readErr, writeErr: tc.writeErr}
			_, err := NewUpdateCommentService(r, tc.filter).UpdateComment(context.Background(), "u1", model.UpdateCommentReq{CommentID: "c1", Content: "hello"})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if (tc.readErr != nil || tc.name == "empty filtered content") && r.writes != 0 {
				t.Fatal("failed request wrote content")
			}
		})
	}
}
