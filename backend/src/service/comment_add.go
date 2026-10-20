package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
)

var (
	ErrInvalidCommentInput       = errors.New("invalid comment input")
	ErrCommentParentNotFound     = data.ErrCommentParentNotFound
	ErrCommentCompensationFailed = data.ErrCommentCompensationFailed
)

type CommentCreator interface {
	GetCommentParent(context.Context, int, string, string) (data.Comment, error)
	CreateComment(context.Context, data.Comment) error
}

type AddCommentService struct {
	repository CommentCreator
	users      CommentUserRepository
	filter     func(string) string
}

// 过滤器由 Handler 注入 tools.FilterSensitiveWords，避免 service 反向依赖 ctrl。
func NewAddCommentService(repository CommentCreator, users CommentUserRepository, filter func(string) string) *AddCommentService {
	if repository == nil {
		repository = data.CommentRepository{}
	}
	if users == nil {
		users = data.AuthRepository{}
	}
	return &AddCommentService{repository: repository, users: users, filter: filter}
}

func (s *AddCommentService) AddComment(ctx context.Context, userID string, req model.AddCommentReq) (model.AddCommentData, error) {
	if strings.TrimSpace(userID) == "" {
		return model.AddCommentData{}, ErrUnauthenticated
	}
	if (req.TargetType != 1 && req.TargetType != 2) || strings.TrimSpace(req.TargetID) == "" || strings.TrimSpace(req.Content) == "" || !utf8.ValidString(req.Content) || (req.ParentID == "" && req.ReplyTo != "") || (req.ParentID != "" && strings.TrimSpace(req.ParentID) == "") || (req.ReplyTo != "" && strings.TrimSpace(req.ReplyTo) == "") {
		return model.AddCommentData{}, ErrInvalidCommentInput
	}
	if s.filter == nil {
		return model.AddCommentData{}, errors.New("comment content filter is not configured")
	}
	content := s.filter(req.Content)
	if strings.TrimSpace(content) == "" {
		return model.AddCommentData{}, ErrInvalidCommentInput
	}
	if req.ParentID != "" {
		if _, err := s.repository.GetCommentParent(ctx, req.TargetType, req.TargetID, req.ParentID); err != nil {
			return model.AddCommentData{}, fmt.Errorf("validate parent comment: %w", err)
		}
	}
	userIDs := []string{userID}
	if req.ReplyTo != "" && req.ReplyTo != userID {
		userIDs = append(userIDs, req.ReplyTo)
	}
	profiles, err := s.users.GetCommentUsers(ctx, userIDs)
	if err != nil {
		return model.AddCommentData{}, fmt.Errorf("load comment author: %w", err)
	}
	author, exists := profiles[userID]
	if !exists {
		return model.AddCommentData{}, ErrUnauthenticated
	}
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return model.AddCommentData{}, fmt.Errorf("generate comment UUID: %w", err)
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	id := fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
	now := time.Now().UTC()
	comment := data.Comment{ID: id, CommentID: id, TargetType: req.TargetType, TargetID: req.TargetID, UserID: userID, Content: content, ParentID: req.ParentID, ReplyTo: req.ReplyTo, Status: 2, CreateTime: now, UpdateTime: now}
	if req.TargetType == 1 {
		comment.QuestionID = req.TargetID
	}
	if err := s.repository.CreateComment(ctx, comment); err != nil {
		return model.AddCommentData{}, fmt.Errorf("create comment: %w", err)
	}
	info := model.CommentInfo{
		CommentID: id, UserID: userID, UserName: author.Username, UserAvatar: author.Avatar,
		Content: content, ParentID: req.ParentID, ReplyTo: req.ReplyTo, ReplyToName: profiles[req.ReplyTo].Username,
		Status: 2, SubComments: []model.CommentInfo{}, CreateTime: formatCommentTime(now), UpdateTime: formatCommentTime(now),
	}
	return model.AddCommentData{CommentID: id, Comment: info}, nil
}
