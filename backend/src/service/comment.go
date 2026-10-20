package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
)

var ErrInvalidCommentQuery = errors.New("invalid comment query")

type CommentRepository interface {
	ListComments(context.Context, data.CommentFilter) ([]data.Comment, int64, error)
	ListCommentChildren(context.Context, data.CommentFilter, []string) ([]data.Comment, error)
	GetLikedComments(context.Context, string, []string) (map[string]bool, error)
}
type CommentUserRepository interface {
	GetCommentUsers(context.Context, []string) (map[string]data.CommentUser, error)
}
type CommentService struct {
	repository CommentRepository
	users      CommentUserRepository
}

func NewCommentService(repository CommentRepository, users CommentUserRepository) *CommentService {
	if repository == nil {
		repository = data.CommentRepository{}
	}
	if users == nil {
		users = data.AuthRepository{}
	}
	return &CommentService{repository: repository, users: users}
}

func ValidateCommentQuery(req model.ListCommentsReq) error {
	validPage := func(page, size int) bool { return page > 0 && size > 0 && int64(page-1) <= (1<<63-1)/int64(size) }
	if (req.TargetType != 1 && req.TargetType != 2) || strings.TrimSpace(req.TargetID) == "" || !validPage(req.Page, req.PageSize) || !validPage(req.SubCommentPage, req.SubCommentSize) || (req.SortBy != "create_time" && req.SortBy != "thumbs_up") || (req.SortOrder != "asc" && req.SortOrder != "desc") {
		return ErrInvalidCommentQuery
	}
	return nil
}

func (s *CommentService) ListComments(ctx context.Context, req model.ListCommentsReq) (model.ListCommentsData, error) {
	if err := ValidateCommentQuery(req); err != nil {
		return model.ListCommentsData{}, err
	}
	f := data.CommentFilter{TargetType: req.TargetType, TargetID: req.TargetID, ParentID: req.ParentID, SortBy: req.SortBy, SortOrder: req.SortOrder, Page: req.Page, PageSize: req.PageSize}
	comments, total, err := s.repository.ListComments(ctx, f)
	if err != nil {
		return model.ListCommentsData{}, fmt.Errorf("list comments: %w", err)
	}
	result := model.ListCommentsData{Total: total, List: make([]model.CommentInfo, 0, len(comments))}
	if len(comments) == 0 {
		return result, nil
	}
	children := make(map[string][]data.Comment)
	counts := make(map[string]int64)
	if req.ParentID == "" {
		parentIDs := make([]string, 0, len(comments))
		for _, c := range comments {
			parentIDs = append(parentIDs, c.CommentID)
		}
		all, err := s.repository.ListCommentChildren(ctx, f, parentIDs)
		if err != nil {
			return model.ListCommentsData{}, fmt.Errorf("load comment children: %w", err)
		}
		for _, child := range all {
			children[child.ParentID] = append(children[child.ParentID], child)
		}
		start := int64(req.SubCommentPage-1) * int64(req.SubCommentSize)
		for _, parentID := range parentIDs {
			group := children[parentID]
			counts[parentID] = int64(len(group))
			if start >= int64(len(group)) {
				children[parentID] = nil
				continue
			}
			end := int64(len(group))
			if int64(req.SubCommentSize) < end-start {
				end = start + int64(req.SubCommentSize)
			}
			children[parentID] = group[start:end]
		}
	}
	displayed := make([]data.Comment, 0, len(comments))
	for _, c := range comments {
		displayed = append(displayed, c)
		displayed = append(displayed, children[c.CommentID]...)
	}
	userIDs := make([]string, 0)
	ids := make([]string, 0, len(displayed))
	seenUsers := map[string]bool{}
	seenComments := map[string]bool{}
	for _, c := range displayed {
		for _, id := range []string{c.UserID, c.ReplyTo} {
			if id != "" && !seenUsers[id] {
				seenUsers[id] = true
				userIDs = append(userIDs, id)
			}
		}
		if !seenComments[c.CommentID] {
			seenComments[c.CommentID] = true
			ids = append(ids, c.CommentID)
		}
	}
	profiles, err := s.users.GetCommentUsers(ctx, userIDs)
	if err != nil {
		return model.ListCommentsData{}, fmt.Errorf("load comment profiles: %w", err)
	}
	likes := map[string]bool{}
	if strings.TrimSpace(req.UserID) != "" {
		likes, err = s.repository.GetLikedComments(ctx, req.UserID, ids)
		if err != nil {
			return model.ListCommentsData{}, fmt.Errorf("load comment likes: %w", err)
		}
	}
	convert := func(c data.Comment) model.CommentInfo {
		return model.CommentInfo{
			CommentID: c.CommentID, UserID: c.UserID,
			UserName: profiles[c.UserID].Username, UserAvatar: profiles[c.UserID].Avatar,
			Content: c.Content, ParentID: c.ParentID, ReplyTo: c.ReplyTo,
			ReplyToName: profiles[c.ReplyTo].Username, Status: c.Status,
			ThumbsUp: c.ThumbsUp, UserLiked: likes[c.CommentID],
			SubComments: []model.CommentInfo{},
			CreateTime:  formatCommentTime(c.CreateTime), UpdateTime: formatCommentTime(c.UpdateTime),
		}
	}
	for _, c := range comments {
		item := convert(c)
		item.SubCommentTotal = counts[c.CommentID]
		for _, child := range children[c.CommentID] {
			item.SubComments = append(item.SubComments, convert(child))
		}
		result.List = append(result.List, item)
	}
	return result, nil
}

func formatCommentTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
