package data

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"reflect"
	"testing"
)

func TestGetQuestionUserState(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("liked and tagged", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, "catalog.user_interactions", mtest.FirstBatch, bson.D{{Key: "_id", Value: "like"}}),
			mtest.CreateCursorResponse(0, "catalog.user_question_tag", mtest.FirstBatch, bson.D{{Key: "tag", Value: 3}}),
		)
		got, err := NewQuestionRepository(mt.Client.Database("catalog")).GetQuestionUserState(context.Background(), "u1", "q1")
		if err != nil || !got.UserLiked || got.UserTag != 3 {
			mt.Fatalf("incorrect state %+v %v", got, err)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 2 {
			mt.Fatalf("expected two lookups: %d", len(events))
		}
		want := []bson.M{
			{"user_id": "u1", "target_id": "q1", "target_type": int32(1), "interaction_type": int32(1), "status": int32(1)},
			{"user_id": "u1", "question_id": "q1"},
		}
		for i, event := range events {
			var filter bson.M
			if err := bson.Unmarshal(event.Command.Lookup("filter").Document(), &filter); err != nil {
				mt.Fatal(err)
			}
			if !reflect.DeepEqual(filter, want[i]) {
				mt.Fatalf("wrong filter %v want %v", filter, want[i])
			}
			collection := []string{"user_interactions", "user_question_tag"}[i]
			if event.Command.Lookup("find").StringValue() != collection {
				mt.Fatalf("wrong collection %s", event.Command)
			}
		}
	})
	mt.Run("no records", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.user_interactions", mtest.FirstBatch), mtest.CreateCursorResponse(0, "catalog.user_question_tag", mtest.FirstBatch))
		got, err := NewQuestionRepository(mt.Client.Database("catalog")).GetQuestionUserState(context.Background(), "u1", "q1")
		if err != nil || got.UserLiked || got.UserTag != 0 {
			mt.Fatalf("missing state %+v %v", got, err)
		}
	})
	for _, stage := range []string{"like", "tag"} {
		mt.Run(stage+" failure", func(mt *mtest.T) {
			if stage == "tag" {
				mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.user_interactions", mtest.FirstBatch))
			}
			mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "query failed"}))
			if _, err := NewQuestionRepository(mt.Client.Database("catalog")).GetQuestionUserState(context.Background(), "u1", "q1"); err == nil {
				mt.Fatal("query error ignored")
			}
		})
	}
}

func TestGetQuestionByIDLoadsAnalysisContent(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("analysis field", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch,
			bson.D{{Key: "question_id", Value: "q1"}, {Key: "analysis_content", Value: "完整解析"}},
		))
		question, err := NewQuestionRepository(mt.Client.Database("catalog")).GetQuestionByID(context.Background(), "q1")
		if err != nil || question.AnalysisContent != "完整解析" {
			mt.Fatalf("analysis not decoded: %+v %v", question, err)
		}
	})
}
