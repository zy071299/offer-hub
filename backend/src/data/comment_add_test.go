package data

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"reflect"
	"testing"
	"time"
)

func TestCommentParentIsNormalRootOfSameTarget(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("parent found", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.comments", mtest.FirstBatch, bson.D{{Key: "comment_id", Value: "root"}, {Key: "parent_id", Value: ""}, {Key: "status", Value: 2}}))
		p, err := NewCommentRepository(mt.Client.Database("catalog")).GetCommentParent(context.Background(), 1, "q1", "root")
		if err != nil || p.CommentID != "root" {
			mt.Fatalf("parent lookup failed %+v %v", p, err)
		}
		var filter bson.M
		bson.Unmarshal(mt.GetStartedEvent().Command.Lookup("filter").Document(), &filter)
		if !reflect.DeepEqual(filter, bson.M{"comment_id": "root", "target_type": int32(1), "target_id": "q1", "parent_id": "", "status": int32(2)}) {
			mt.Fatalf("unsafe parent filter %v", filter)
		}
	})
	mt.Run("missing parent", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.comments", mtest.FirstBatch))
		if _, err := NewCommentRepository(mt.Client.Database("catalog")).GetCommentParent(context.Background(), 1, "q1", "missing"); !errors.Is(err, ErrCommentParentNotFound) {
			mt.Fatalf("incorrect error %v", err)
		}
	})
}
func TestCreateCommentUpdatesParentAndCompensatesFailure(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	c := Comment{ID: "new", CommentID: "new", TargetType: 1, TargetID: "q1", QuestionID: "q1", ParentID: "root", UserID: "u1", Content: "filtered", Status: 2, CreateTime: time.Now().UTC(), UpdateTime: time.Now().UTC()}
	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}), mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		if err := NewCommentRepository(mt.Client.Database("catalog")).CreateComment(context.Background(), c); err != nil {
			mt.Fatal(err)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 2 || events[0].CommandName != "insert" || events[1].CommandName != "update" {
			mt.Fatal("wrong mutation sequence")
		}
		insert := events[0].Command.Lookup("documents").Array().Index(0).Value().Document()
		if insert.Lookup("_id").StringValue() != "new" || insert.Lookup("content").StringValue() != "filtered" || insert.Lookup("status").Int32() != 2 {
			mt.Fatalf("wrong stored doc %s", insert)
		}
		update := events[1].Command.Lookup("updates").Array().Index(0).Value().Document()
		var filter bson.M
		bson.Unmarshal(update.Lookup("q").Document(), &filter)
		if !reflect.DeepEqual(filter, bson.M{"comment_id": "root", "target_type": int32(1), "target_id": "q1", "parent_id": "", "status": int32(2)}) {
			mt.Fatalf("unsafe update %v", filter)
		}
		if update.Lookup("u").Document().Lookup("$inc").Document().Lookup("child_count").Int32() != 1 {
			mt.Fatal("parent not incremented atomically")
		}
	})
	for _, mode := range []string{"missing parent", "update failed", "rollback failed", "insert failed", "top level"} {
		mt.Run(mode, func(mt *mtest.T) {
			comment := c
			if mode == "insert failed" {
				mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "insert failed"}))
			} else {
				mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
				if mode == "top level" {
					comment.ParentID = ""
				} else {
					if mode == "missing parent" {
						mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 0}, bson.E{Key: "nModified", Value: 0}))
					} else {
						mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "update failed"}))
					}
					if mode == "rollback failed" {
						mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "delete failed"}))
					} else {
						mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}))
					}
				}
			}
			err := NewCommentRepository(mt.Client.Database("catalog")).CreateComment(context.Background(), comment)
			if mode == "top level" {
				if err != nil {
					mt.Fatal(err)
				}
			} else if err == nil {
				mt.Fatal("write failure reported success")
			}
			if mode == "missing parent" && !errors.Is(err, ErrCommentParentNotFound) {
				mt.Fatalf("lost missing-parent classification %v", err)
			}
			if mode == "rollback failed" && !errors.Is(err, ErrCommentCompensationFailed) {
				mt.Fatalf("compensation error classification lost: %v", err)
			}
			events := mt.GetAllStartedEvents()
			if mode == "top level" || mode == "insert failed" {
				if len(events) != 1 {
					mt.Fatal("unexpected parent update")
				}
				return
			}
			if len(events) != 3 || events[2].CommandName != "delete" {
				mt.Fatal("missing compensation delete")
			}
			var filter bson.M
			bson.Unmarshal(events[2].Command.Lookup("deletes").Array().Index(0).Value().Document().Lookup("q").Document(), &filter)
			if !reflect.DeepEqual(filter, bson.M{"_id": "new", "comment_id": "new", "user_id": "u1"}) {
				mt.Fatalf("unsafe compensation %v", filter)
			}
		})
	}
}
