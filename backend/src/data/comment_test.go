package data

import (
	"context"
	"database/sql/driver"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"reflect"
	"strings"
	"testing"
)

func TestCommentMongoQueries(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("paged parent list", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.comments", mtest.FirstBatch, bson.D{{Key: "n", Value: int64(7)}}), mtest.CreateCursorResponse(0, "catalog.comments", mtest.FirstBatch, bson.D{{Key: "comment_id", Value: "c1"}, {Key: "status", Value: 2}}))
		r := NewCommentRepository(mt.Client.Database("catalog"))
		items, total, err := r.ListComments(context.Background(), CommentFilter{TargetType: 1, TargetID: "q1", Page: 2, PageSize: 3, SortBy: "thumbs_up", SortOrder: "desc"})
		if err != nil || total != 7 || len(items) != 1 {
			mt.Fatalf("incorrect list %+v %d %v", items, total, err)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 2 {
			mt.Fatalf("unexpected queries %d", len(events))
		}
		cmd := events[1].Command
		var filter bson.M
		bson.Unmarshal(cmd.Lookup("filter").Document(), &filter)
		if !reflect.DeepEqual(filter, bson.M{"target_type": int32(1), "target_id": "q1", "parent_id": "", "status": int32(2)}) {
			mt.Fatalf("wrong filter %v", filter)
		}
		if cmd.Lookup("skip").Int64() != 3 || cmd.Lookup("limit").Int64() != 3 {
			mt.Fatalf("wrong pagination %s", cmd)
		}
		var sort bson.D
		bson.Unmarshal(cmd.Lookup("sort").Document(), &sort)
		if !reflect.DeepEqual(sort, bson.D{{Key: "thumbs_up", Value: int32(-1)}, {Key: "comment_id", Value: int32(1)}}) {
			mt.Fatalf("wrong sort %v", sort)
		}
	})
	mt.Run("batch children and likes", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.comments", mtest.FirstBatch), mtest.CreateCursorResponse(0, "catalog.user_interactions", mtest.FirstBatch, bson.D{{Key: "target_id", Value: "c1"}}))
		r := NewCommentRepository(mt.Client.Database("catalog"))
		children, err := r.ListCommentChildren(context.Background(), CommentFilter{TargetType: 1, TargetID: "q1"}, []string{"c1", "c2"})
		if err != nil || children == nil {
			mt.Fatal(err)
		}
		likes, err := r.GetLikedComments(context.Background(), "viewer", []string{"c1", "s1"})
		if err != nil || !likes["c1"] || likes["s1"] {
			mt.Fatalf("incorrect likes %v %v", likes, err)
		}
		events := mt.GetAllStartedEvents()
		var childrenFilter, likeFilter bson.M
		bson.Unmarshal(events[0].Command.Lookup("filter").Document(), &childrenFilter)
		bson.Unmarshal(events[1].Command.Lookup("filter").Document(), &likeFilter)
		if !reflect.DeepEqual(childrenFilter, bson.M{"target_type": int32(1), "target_id": "q1", "status": int32(2), "parent_id": bson.M{"$in": bson.A{"c1", "c2"}}}) {
			mt.Fatalf("wrong children filter %v", childrenFilter)
		}
		if !reflect.DeepEqual(likeFilter, bson.M{"user_id": "viewer", "target_type": int32(3), "interaction_type": int32(1), "status": int32(1), "target_id": bson.M{"$in": bson.A{"c1", "s1"}}}) {
			mt.Fatalf("wrong likes filter %v", likeFilter)
		}
		var sort bson.D
		bson.Unmarshal(events[0].Command.Lookup("sort").Document(), &sort)
		if !reflect.DeepEqual(sort, bson.D{{Key: "create_time", Value: int32(1)}, {Key: "comment_id", Value: int32(1)}}) {
			mt.Fatalf("wrong child sort %v", sort)
		}
	})
	mt.Run("Mongo errors", func(mt *mtest.T) {
		r := NewCommentRepository(mt.Client.Database("catalog"))
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "failed"}))
		if _, _, err := r.ListComments(context.Background(), CommentFilter{TargetType: 1, TargetID: "q1", Page: 1, PageSize: 20, SortBy: "create_time", SortOrder: "desc"}); err == nil {
			mt.Fatal("count failure ignored")
		}
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "failed"}))
		if _, err := r.ListCommentChildren(context.Background(), CommentFilter{TargetType: 1, TargetID: "q1"}, []string{"c1"}); err == nil {
			mt.Fatal("child failure ignored")
		}
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "failed"}))
		if _, err := r.GetLikedComments(context.Background(), "u", []string{"c1"}); err == nil {
			mt.Fatal("like failure ignored")
		}
	})
}

type commentProfileRows struct{ authUserRows }

func (*commentProfileRows) Columns() []string { return []string{"user_id", "username", "avatar"} }
func TestCommentUserLookupBatchesWithoutPrivateFields(t *testing.T) {
	conn := &authSQLConn{rows: &commentProfileRows{authUserRows{values: []driver.Value{"u1", "cat", "avatar"}}}}
	got, err := authTestRepository(t, conn).GetCommentUsers(context.Background(), []string{"u1", "u2"})
	if err != nil || got["u1"].Username != "cat" || got["u1"].Avatar != "avatar" || len(got) != 1 {
		t.Fatalf("incorrect profiles %v %v", got, err)
	}
	if !strings.Contains(conn.query, "SELECT `user_id`,`username`,`avatar` FROM `t_user_info` WHERE user_id IN (?,?)") || len(conn.args) != 2 {
		t.Fatalf("incorrect profile query %s", conn.query)
	}
	failure := errors.New("SQL failed")
	conn.err = failure
	if _, err := authTestRepository(t, conn).GetCommentUsers(context.Background(), []string{"u1"}); !errors.Is(err, failure) {
		t.Fatalf("SQL error lost %v", err)
	}
}
