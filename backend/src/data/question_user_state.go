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

type QuestionUserState struct {
	UserLiked bool
	UserTag   int
}

func (r QuestionRepository) GetQuestionUserState(ctx context.Context, userID, questionID string) (QuestionUserState, error) {
	db, err := r.database()
	if err != nil {
		return QuestionUserState{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result := QuestionUserState{}
	filter := bson.D{
		{Key: "user_id", Value: userID}, {Key: "target_id", Value: questionID},
		{Key: "target_type", Value: 1}, {Key: "interaction_type", Value: 1}, {Key: "status", Value: 1},
	}
	var liked bson.M
	err = db.Collection("user_interactions").FindOne(ctx, filter, options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 1}})).Decode(&liked)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return QuestionUserState{}, fmt.Errorf("find user like: %w", err)
	}
	result.UserLiked = err == nil
	var tag struct {
		Tag int `bson:"tag"`
	}
	err = db.Collection("user_question_tag").FindOne(ctx, bson.D{{Key: "user_id", Value: userID}, {Key: "question_id", Value: questionID}}, options.FindOne().SetProjection(bson.D{{Key: "tag", Value: 1}, {Key: "_id", Value: 0}})).Decode(&tag)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return QuestionUserState{}, fmt.Errorf("find user question tag: %w", err)
	}
	result.UserTag = tag.Tag
	return result, nil
}
