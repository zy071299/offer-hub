package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
)

type CommentUpdaterRepository interface {
	GetCommentByID(context.Context, string) (data.Comment, error)
	UpdateCommentContent(context.Context, string, string, string) error
}

type UpdateCommentService struct {
	repository CommentUpdaterRepository
	filter     func(string) string
}

// 由 Handler 注入 tools.FilterSensitiveWords，保持 service 与 ctrl 的依赖方向。
func NewUpdateCommentService(repository CommentUpdaterRepository, filter func(string) string) *UpdateCommentService {
	if repository == nil {
		repository = data.CommentRepository{}
	}
	return &UpdateCommentService{repository: repository, filter: filter}
}

func (s *UpdateCommentService) UpdateComment(ctx context.Context, userID string, req model.UpdateCommentReq) (model.UpdateCommentData, error) {
	if strings.TrimSpace(userID) == "" {
		return model.UpdateCommentData{}, ErrUnauthenticated
	}
	if strings.TrimSpace(req.CommentID) == "" || strings.TrimSpace(req.Content) == "" || !utf8.ValidString(req.Content) {
		return model.UpdateCommentData{}, ErrInvalidCommentInput
	}
	comment, err := s.repository.GetCommentByID(ctx, req.CommentID)
	if err != nil {
		return model.UpdateCommentData{}, fmt.Errorf("load comment for update: %w", err)
	}
	if comment.UserID != userID {
		return model.UpdateCommentData{}, ErrCommentForbidden
	}
	if comment.Status == 5 {
		return model.UpdateCommentData{}, ErrCommentNotFound
	}
	if s.filter == nil {
		return model.UpdateCommentData{}, errors.New("comment content filter is not configured")
	}
	content := s.filter(req.Content)
	if strings.TrimSpace(content) == "" {
		return model.UpdateCommentData{}, ErrInvalidCommentInput
	}
	if err := s.repository.UpdateCommentContent(ctx, req.CommentID, userID, content); err != nil {
		return model.UpdateCommentData{}, fmt.Errorf("update comment: %w", err)
	}
	return model.UpdateCommentData{CommentID: req.CommentID}, nil
}
