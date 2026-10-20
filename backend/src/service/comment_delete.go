package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"offer-hub/backend/src/data"
)

var (
	ErrCommentNotFound  = data.ErrCommentNotFound
	ErrCommentForbidden = errors.New("comment belongs to another user")
)

type CommentDeleterRepository interface {
	GetCommentByID(context.Context, string) (data.Comment, error)
	SoftDeleteComment(context.Context, string, string) error
}

type DeleteCommentService struct{ repository CommentDeleterRepository }

func NewDeleteCommentService(repository CommentDeleterRepository) *DeleteCommentService {
	if repository == nil {
		repository = data.CommentRepository{}
	}
	return &DeleteCommentService{repository: repository}
}

func (s *DeleteCommentService) DeleteComment(ctx context.Context, userID, commentID string) error {
	if strings.TrimSpace(userID) == "" {
		return ErrUnauthenticated
	}
	if strings.TrimSpace(commentID) == "" {
		return ErrInvalidCommentInput
	}
	comment, err := s.repository.GetCommentByID(ctx, commentID)
	if err != nil {
		return fmt.Errorf("load comment for deletion: %w", err)
	}
	if comment.UserID != userID {
		return ErrCommentForbidden
	}
	if comment.Status == 5 {
		return nil
	}
	if err := s.repository.SoftDeleteComment(ctx, commentID, userID); err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}
