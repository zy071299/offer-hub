package data

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// UpdateCommentContent 在写入时再次检查作者和删除状态，避免查询后的并发删除竞态。
func (r CommentRepository) UpdateCommentContent(ctx context.Context, commentID, userID, content string) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result, err := db.Collection("comments").UpdateOne(ctx,
		bson.D{{Key: "comment_id", Value: commentID}, {Key: "user_id", Value: userID}, {Key: "status", Value: bson.D{{Key: "$ne", Value: 5}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "content", Value: content}, {Key: "update_time", Value: time.Now().UTC()}}}},
	)
	if err != nil {
		return fmt.Errorf("update comment content: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrCommentNotFound
	}
	return nil
}
