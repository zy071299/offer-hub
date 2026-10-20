package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Comment struct {
	ID         string    `bson:"_id"`
	CommentID  string    `bson:"comment_id"`
	TargetType int       `bson:"target_type"`
	TargetID   string    `bson:"target_id"`
	QuestionID string    `bson:"question_id"`
	UserID     string    `bson:"user_id"`
	Content    string    `bson:"content"`
	ParentID   string    `bson:"parent_id"`
	ReplyTo    string    `bson:"reply_to"`
	ChildCount int       `bson:"child_count"`
	ThumbsUp   int       `bson:"thumbs_up"`
	ViewCount  int       `bson:"view_count"`
	Status     int       `bson:"status"`
	CreateTime time.Time `bson:"create_time"`
	UpdateTime time.Time `bson:"update_time"`
}

type CommentFilter struct {
	TargetType                            int
	TargetID, ParentID, SortBy, SortOrder string
	Page, PageSize                        int
}

type CommentRepository struct{ db *mongo.Database }

func NewCommentRepository(db *mongo.Database) *CommentRepository { return &CommentRepository{db: db} }

func (r CommentRepository) database() (*mongo.Database, error) {
	return (QuestionRepository{db: r.db}).database()
}

func commentTargetFilter(f CommentFilter) bson.D {
	return bson.D{{Key: "target_type", Value: f.TargetType}, {Key: "target_id", Value: f.TargetID}, {Key: "status", Value: 2}}
}

func (r CommentRepository) ListComments(ctx context.Context, f CommentFilter) ([]Comment, int64, error) {
	items := make([]Comment, 0)
	if f.Page < 1 || f.PageSize < 1 || int64(f.Page-1) > (1<<63-1)/int64(f.PageSize) {
		return nil, 0, errors.New("invalid comment pagination")
	}
	if f.SortBy != "create_time" && f.SortBy != "thumbs_up" {
		return nil, 0, errors.New("invalid comment sort field")
	}
	direction := 1
	if f.SortOrder == "desc" {
		direction = -1
	} else if f.SortOrder != "asc" {
		return nil, 0, errors.New("invalid comment sort direction")
	}
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	filter := append(commentTargetFilter(f), bson.E{Key: "parent_id", Value: f.ParentID})
	total, err := db.Collection("comments").CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
	}
	if total == 0 {
		return items, 0, nil
	}
	cursor, err := db.Collection("comments").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: f.SortBy, Value: direction}, {Key: "comment_id", Value: 1}}).SetSkip(int64(f.Page-1)*int64(f.PageSize)).SetLimit(int64(f.PageSize)))
	if err != nil {
		return nil, 0, fmt.Errorf("find comments: %w", err)
	}
	if err = cursor.All(ctx, &items); err != nil {
		return nil, 0, fmt.Errorf("decode comments: %w", err)
	}
	return items, total, nil
}

// ListCommentChildren 一次查询当前页所有父评论的正常子评论，供分组分页和计数。
func (r CommentRepository) ListCommentChildren(ctx context.Context, f CommentFilter, parentIDs []string) ([]Comment, error) {
	items := make([]Comment, 0)
	if len(parentIDs) == 0 {
		return items, nil
	}
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	filter := append(commentTargetFilter(f), bson.E{Key: "parent_id", Value: bson.D{{Key: "$in", Value: parentIDs}}})
	cursor, err := db.Collection("comments").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "create_time", Value: 1}, {Key: "comment_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find child comments: %w", err)
	}
	if err = cursor.All(ctx, &items); err != nil {
		return nil, fmt.Errorf("decode child comments: %w", err)
	}
	return items, nil
}

func (r CommentRepository) GetLikedComments(ctx context.Context, userID string, ids []string) (map[string]bool, error) {
	result := make(map[string]bool)
	if userID == "" || len(ids) == 0 {
		return result, nil
	}
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	filter := bson.D{{Key: "user_id", Value: userID}, {Key: "target_type", Value: 3}, {Key: "interaction_type", Value: 1}, {Key: "status", Value: 1}, {Key: "target_id", Value: bson.D{{Key: "$in", Value: ids}}}}
	cursor, err := db.Collection("user_interactions").Find(ctx, filter, options.Find().SetProjection(bson.D{{Key: "target_id", Value: 1}, {Key: "_id", Value: 0}}))
	if err != nil {
		return nil, fmt.Errorf("find comment likes: %w", err)
	}
	var likes []struct {
		TargetID string `bson:"target_id"`
	}
	if err = cursor.All(ctx, &likes); err != nil {
		return nil, fmt.Errorf("decode comment likes: %w", err)
	}
	for _, like := range likes {
		result[like.TargetID] = true
	}
	return result, nil
}

type CommentUser struct {
	UserID   string `gorm:"column:user_id"`
	Username string
	Avatar   string
}

func (r AuthRepository) GetCommentUsers(ctx context.Context, ids []string) (map[string]CommentUser, error) {
	result := make(map[string]CommentUser)
	if len(ids) == 0 {
		return result, nil
	}
	db, err := r.database(ctx)
	if err != nil {
		return nil, err
	}
	var users []CommentUser
	if err = db.Table("t_user_info").Select("user_id", "username", "avatar").Where("user_id IN ?", ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("load comment users: %w", err)
	}
	for _, user := range users {
		result[user.UserID] = user
	}
	return result, nil
}
