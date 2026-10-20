package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrCommentNotFound = errors.New("comment not found")

// GetCommentByID 不过滤状态，让业务层区分作者权限与本人重复删除。
func (r CommentRepository) GetCommentByID(ctx context.Context, commentID string) (Comment, error) {
	db, err := r.database()
	if err != nil {
		return Comment{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var comment Comment
	err = db.Collection("comments").FindOne(ctx, bson.D{{Key: "comment_id", Value: commentID}}).Decode(&comment)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Comment{}, ErrCommentNotFound
	}
	if err != nil {
		return Comment{}, fmt.Errorf("get comment by id: %w", err)
	}
	return comment, nil
}

// SoftDeleteComment 在写入条件中再次限定作者，防止更新其他用户的评论。
func (r CommentRepository) SoftDeleteComment(ctx context.Context, commentID, userID string) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result, err := db.Collection("comments").UpdateOne(ctx,
		bson.D{{Key: "comment_id", Value: commentID}, {Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: 5}, {Key: "update_time", Value: time.Now().UTC()}}}},
	)
	if err != nil {
		return fmt.Errorf("soft delete comment: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrCommentNotFound
	}
	return nil
}
