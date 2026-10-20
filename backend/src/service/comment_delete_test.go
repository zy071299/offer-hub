package service

import (
	"context"
	"errors"
	"offer-hub/backend/src/data"
	"testing"
)

type commentDeleteRepo struct {
	comment                        data.Comment
	readErr, writeErr              error
	readID, deletedID, deletedUser string
	writes                         int
}

func (r *commentDeleteRepo) GetCommentByID(_ context.Context, id string) (data.Comment, error) {
	r.readID = id
	return r.comment, r.readErr
}
func (r *commentDeleteRepo) SoftDeleteComment(_ context.Context, id, user string) error {
	r.writes++
	r.deletedID = id
	r.deletedUser = user
	return r.writeErr
}
func TestDeleteCommentOwnershipAndIdempotency(t *testing.T) {
	for _, tc := range []struct {
		name, user string
		status     int
		want       error
		writes     int
	}{
		{"owner", "u1", 2, nil, 1}, {"other user", "u2", 2, ErrCommentForbidden, 0},
		{"already deleted", "u1", 5, nil, 0}, {"deleted by other author", "u2", 5, ErrCommentForbidden, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &commentDeleteRepo{comment: data.Comment{CommentID: "c1", UserID: "u1", Status: tc.status}}
			err := NewDeleteCommentService(r).DeleteComment(context.Background(), tc.user, "c1")
			if !errors.Is(err, tc.want) || r.writes != tc.writes || r.readID != "c1" {
				t.Fatalf("error=%v writes=%d", err, r.writes)
			}
			if tc.writes > 0 && (r.deletedID != "c1" || r.deletedUser != "u1") {
				t.Fatal("write lost ownership condition")
			}
		})
	}
}
func TestDeleteCommentValidationAndErrors(t *testing.T) {
	for _, tc := range []struct {
		user, id string
		want     error
	}{{"", "c1", ErrUnauthenticated}, {"u1", "  ", ErrInvalidCommentInput}} {
		r := &commentDeleteRepo{}
		if err := NewDeleteCommentService(r).DeleteComment(context.Background(), tc.user, tc.id); !errors.Is(err, tc.want) || r.readID != "" || r.writes != 0 {
			t.Fatalf("invalid input accessed storage: %v", err)
		}
	}
	failure := errors.New("database failed")
	for _, tc := range []struct{ readErr, writeErr, want error }{
		{data.ErrCommentNotFound, nil, data.ErrCommentNotFound}, {failure, nil, failure}, {nil, failure, failure},
	} {
		r := &commentDeleteRepo{comment: data.Comment{UserID: "u1", Status: 2}, readErr: tc.readErr, writeErr: tc.writeErr}
		if err := NewDeleteCommentService(r).DeleteComment(context.Background(), "u1", "c1"); !errors.Is(err, tc.want) {
			t.Fatalf("error lost %v", err)
		}
		if tc.readErr != nil && r.writes != 0 {
			t.Fatal("failed read reached update")
		}
	}
}
