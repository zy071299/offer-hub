package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrCommentParentNotFound     = errors.New("normal top-level parent comment not found")
	ErrCommentCompensationFailed = errors.New("comment insertion compensation failed")
)

func commentParentFilter(targetType int, targetID, parentID string) bson.D {
	return bson.D{{Key: "comment_id", Value: parentID}, {Key: "target_type", Value: targetType}, {Key: "target_id", Value: targetID}, {Key: "parent_id", Value: ""}, {Key: "status", Value: 2}}
}

func (r CommentRepository) GetCommentParent(ctx context.Context, targetType int, targetID, parentID string) (Comment, error) {
	db, err := r.database()
	if err != nil {
		return Comment{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var parent Comment
	err = db.Collection("comments").FindOne(ctx, commentParentFilter(targetType, targetID, parentID)).Decode(&parent)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Comment{}, ErrCommentParentNotFound
	}
	if err != nil {
		return Comment{}, fmt.Errorf("load comment parent: %w", err)
	}
	return parent, nil
}

// CreateComment 插入评论并更新父计数；父更新失败时尝试补偿删除新评论。
// 两次写入并非事务，补偿失败会显式返回错误，不能保证跨文档原子性。
func (r CommentRepository) CreateComment(ctx context.Context, comment Comment) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	collection := db.Collection("comments")
	if _, err = collection.InsertOne(writeCtx, comment); err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	if comment.ParentID == "" {
		return nil
	}
	updated, err := collection.UpdateOne(writeCtx, commentParentFilter(comment.TargetType, comment.TargetID, comment.ParentID), bson.D{
		{Key: "$inc", Value: bson.D{{Key: "child_count", Value: 1}}},
		{Key: "$set", Value: bson.D{{Key: "update_time", Value: comment.UpdateTime}}},
	})
	if err == nil && updated.MatchedCount > 0 {
		return nil
	}
	parentErr := ErrCommentParentNotFound
	if err != nil {
		parentErr = fmt.Errorf("increment parent child count: %w", err)
	}
	// 请求取消或写入超时不应跳过清理；只删除本次生成的评论。
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cleanupCancel()
	_, cleanupErr := collection.DeleteOne(cleanupCtx, bson.D{{Key: "_id", Value: comment.ID}, {Key: "comment_id", Value: comment.CommentID}, {Key: "user_id", Value: comment.UserID}})
	if cleanupErr != nil {
		return errors.Join(ErrCommentCompensationFailed, parentErr, fmt.Errorf("remove inserted comment: %w", cleanupErr))
	}
	return parentErr
}
