package data

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"reflect"
	"testing"
)

func TestCommentDeleteQueries(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("lookup and owned update", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.comments", mtest.FirstBatch, bson.D{{Key: "comment_id", Value: "c1"}, {Key: "user_id", Value: "u1"}, {Key: "status", Value: 2}}), mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		r := NewCommentRepository(mt.Client.Database("catalog"))
		c, err := r.GetCommentByID(context.Background(), "c1")
		if err != nil || c.UserID != "u1" {
			mt.Fatalf("lookup %+v %v", c, err)
		}
		if err := r.SoftDeleteComment(context.Background(), "c1", "u1"); err != nil {
			mt.Fatal(err)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 2 {
			mt.Fatal("wrong query count")
		}
		var lookup, filter bson.M
		bson.Unmarshal(events[0].Command.Lookup("filter").Document(), &lookup)
		if !reflect.DeepEqual(lookup, bson.M{"comment_id": "c1"}) {
			mt.Fatalf("wrong lookup %v", lookup)
		}
		update := events[1].Command.Lookup("updates").Array().Index(0).Value().Document()
		bson.Unmarshal(update.Lookup("q").Document(), &filter)
		if !reflect.DeepEqual(filter, bson.M{"comment_id": "c1", "user_id": "u1"}) {
			mt.Fatalf("unsafe delete filter %v", filter)
		}
		set := update.Lookup("u").Document().Lookup("$set").Document()
		if set.Lookup("status").Int32() != 5 || set.Lookup("update_time").DateTime() <= 0 {
			mt.Fatalf("wrong soft delete %s", set)
		}
	})
	mt.Run("missing read and update", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.comments", mtest.FirstBatch), mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}, bson.E{Key: "nModified", Value: 0}))
		r := NewCommentRepository(mt.Client.Database("catalog"))
		if _, err := r.GetCommentByID(context.Background(), "missing"); !errors.Is(err, ErrCommentNotFound) {
			mt.Fatalf("incorrect read error %v", err)
		}
		if err := r.SoftDeleteComment(context.Background(), "missing", "u1"); !errors.Is(err, ErrCommentNotFound) {
			mt.Fatalf("incorrect update error %v", err)
		}
	})
	mt.Run("command errors", func(mt *mtest.T) {
		r := NewCommentRepository(mt.Client.Database("catalog"))
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "read failed"}))
		if _, err := r.GetCommentByID(context.Background(), "c1"); err == nil || errors.Is(err, ErrCommentNotFound) {
			mt.Fatalf("read failure hidden %v", err)
		}
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "update failed"}))
		if err := r.SoftDeleteComment(context.Background(), "c1", "u1"); err == nil || errors.Is(err, ErrCommentNotFound) {
			mt.Fatalf("update failure hidden %v", err)
		}
	})
}
