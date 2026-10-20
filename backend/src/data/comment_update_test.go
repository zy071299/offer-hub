package data

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"reflect"
	"testing"
)

func TestUpdateCommentContentIsOwnedAndNotDeleted(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("update", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		if err := NewCommentRepository(mt.Client.Database("catalog")).UpdateCommentContent(context.Background(), "c1", "u1", "filtered🙂"); err != nil {
			mt.Fatal(err)
		}
		command := mt.GetStartedEvent().Command.Lookup("updates").Array().Index(0).Value().Document()
		var filter bson.M
		bson.Unmarshal(command.Lookup("q").Document(), &filter)
		if !reflect.DeepEqual(filter, bson.M{"comment_id": "c1", "user_id": "u1", "status": bson.M{"$ne": int32(5)}}) {
			mt.Fatalf("unsafe update filter %v", filter)
		}
		update := command.Lookup("u").Document()
		set := update.Lookup("$set").Document()
		if set.Lookup("content").StringValue() != "filtered🙂" || set.Lookup("update_time").DateTime() <= 0 {
			mt.Fatalf("wrong fields %s", update)
		}
		if values, _ := set.Elements(); len(values) != 2 {
			mt.Fatalf("unexpected fields modified %s", set)
		}
	})
	mt.Run("zero match", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}, bson.E{Key: "nModified", Value: 0}))
		if err := NewCommentRepository(mt.Client.Database("catalog")).UpdateCommentContent(context.Background(), "c1", "u1", "text"); !errors.Is(err, ErrCommentNotFound) {
			mt.Fatalf("concurrent deletion ignored %v", err)
		}
	})
	mt.Run("failure", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "write failed"}))
		if err := NewCommentRepository(mt.Client.Database("catalog")).UpdateCommentContent(context.Background(), "c1", "u1", "text"); err == nil || errors.Is(err, ErrCommentNotFound) {
			mt.Fatalf("database error hidden %v", err)
		}
	})
}
