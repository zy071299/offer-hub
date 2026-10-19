package data

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"reflect"
	"testing"
)

func TestQuestionCatalogMongoCommands(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("filtered catalog", func(mt *mtest.T) {
		db := mt.Client.Database("catalog")
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, "catalog.question_bank_series", mtest.FirstBatch, bson.D{{Key: "series_id", Value: "s1"}, {Key: "series_name", Value: "数据库"}, {Key: "job_name", Value: "backend"}, {Key: "order", Value: int64(1)}}),
			mtest.CreateCursorResponse(0, "catalog.question_bank", mtest.FirstBatch, bson.D{{Key: "bank_id", Value: "b1"}, {Key: "series_id", Value: "s1"}, {Key: "job_name", Value: "backend"}, {Key: "order", Value: int64(2)}}),
			mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch, bson.D{{Key: "_id", Value: "b1"}, {Key: "count", Value: int64(3)}}),
		)
		got, err := NewQuestionRepository(db).LoadQuestionCatalog(context.Background(), "backend")
		if err != nil {
			mt.Fatal(err)
		}
		if len(got.Series) != 1 || got.Series[0].SeriesID != "s1" || len(got.Banks) != 1 || got.Banks[0].BankID != "b1" || got.Counts["b1"] != 3 {
			mt.Fatalf("incorrect decoding: %+v", got)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 3 {
			mt.Fatalf("want two finds and one aggregate, got %d", len(events))
		}
		for _, ev := range events[:2] {
			if ev.Command.Lookup("filter").Document().Lookup("job_name").StringValue() != "backend" {
				mt.Fatal("missing job_name filter")
			}
			if ev.Command.Lookup("sort").Document().Lookup("order").AsInt64() != 1 {
				mt.Fatal("missing ascending order")
			}
		}
		if events[2].Command.Lookup("aggregate").StringValue() != "question" {
			mt.Fatal("wrong count collection")
		}
		values, err := events[2].Command.Lookup("pipeline").Array().Values()
		if err != nil {
			mt.Fatal(err)
		}
		first := values[0].Document().Lookup("$match").Document()
		if first.Lookup("status").AsInt64() != 1 {
			mt.Fatal("counts must only include normal questions")
		}
		ids, err := first.Lookup("bank_list").Document().Lookup("$in").Array().Values()
		if err != nil {
			mt.Fatal(err)
		}
		if len(ids) != 1 || ids[0].StringValue() != "b1" {
			mt.Fatal("count query not limited to returned banks")
		}
		// 检查按题目和题库去重后再计数，防止重复 bank_list 元素重复计数。
		var group bson.M
		if err := bson.Unmarshal(values[3].Document(), &group); err != nil {
			mt.Fatal(err)
		}
		want := bson.M{"$group": bson.M{"_id": bson.M{"question": "$_id", "bank_id": "$bank_list"}}}
		if !reflect.DeepEqual(group, want) {
			mt.Fatalf("missing per-question deduplication: %v", group)
		}
	})
	mt.Run("empty", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question_bank_series", mtest.FirstBatch))
		got, err := NewQuestionRepository(mt.Client.Database("catalog")).LoadQuestionCatalog(context.Background(), "")
		if err != nil || len(got.Series) != 0 || len(mt.GetAllStartedEvents()) != 1 {
			mt.Fatalf("empty catalog: %+v, %v", got, err)
		}
	})
	mt.Run("database failure", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "query failed"}))
		if _, err := NewQuestionRepository(mt.Client.Database("catalog")).LoadQuestionCatalog(context.Background(), ""); err == nil {
			mt.Fatal("expected database error")
		}
	})
}

func TestFilterQuestionsMongoCommands(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("all filters and pagination", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch, bson.D{{Key: "n", Value: int64(31)}}),
			mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch, bson.D{{Key: "question_id", Value: "q1"}, {Key: "bank_list", Value: bson.A{"b1"}}, {Key: "status", Value: 1}, {Key: "content", Value: "完整内容"}, {Key: "order", Value: int64(8)}}),
		)
		questions, total, err := NewQuestionRepository(mt.Client.Database("catalog")).FilterQuestions(context.Background(), QuestionFilter{
			BankID: "b1", Keyword: "Go.*", Difficulty: 2, Tags: []string{"Go", "数据库"}, JobName: "backend", Page: 3, PageSize: 10, SortBy: "view_count", SortOrder: "asc",
		})
		if err != nil {
			mt.Fatal(err)
		}
		if total != 31 || len(questions) != 1 || questions[0].QuestionID != "q1" || questions[0].Content != "完整内容" || questions[0].Order != 8 {
			mt.Fatalf("incorrect result: %v %d", questions, total)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 2 {
			mt.Fatalf("expected count and find, got %d", len(events))
		}
		find := events[1].Command
		var filter bson.M
		if err := bson.Unmarshal(find.Lookup("filter").Document(), &filter); err != nil {
			mt.Fatal(err)
		}
		want := bson.M{"status": int32(1), "bank_list": "b1", "difficulty": int32(2), "tags": bson.M{"$all": bson.A{"Go", "数据库"}}, "job_name": "backend", "$or": bson.A{
			bson.M{"title": primitive.Regex{Pattern: `Go\.\*`, Options: "i"}}, bson.M{"content": primitive.Regex{Pattern: `Go\.\*`, Options: "i"}},
		}}
		if !reflect.DeepEqual(filter, want) {
			mt.Fatalf("unexpected filter: %#v", filter)
		}
		if find.Lookup("skip").AsInt64() != 20 || find.Lookup("limit").AsInt64() != 10 {
			mt.Fatal("incorrect skip or limit")
		}
		sort := find.Lookup("sort").Document()
		if sort.Lookup("view_count").AsInt64() != 1 || sort.Lookup("question_id").AsInt64() != 1 {
			mt.Fatal("incorrect stable sort")
		}
		pipeline, _ := events[0].Command.Lookup("pipeline").Array().Values()
		var countFilter bson.M
		if err := bson.Unmarshal(pipeline[0].Document().Lookup("$match").Document(), &countFilter); err != nil {
			mt.Fatal(err)
		}
		if !reflect.DeepEqual(countFilter, filter) {
			mt.Fatal("count and find must use identical filters")
		}
	})
	mt.Run("empty defaults", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch), mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch))
		questions, total, err := NewQuestionRepository(mt.Client.Database("catalog")).FilterQuestions(context.Background(), QuestionFilter{})
		if err != nil || total != 0 || questions == nil || len(questions) != 0 {
			mt.Fatalf("empty result: %v %d %v", questions, total, err)
		}
		events := mt.GetAllStartedEvents()
		find := events[1].Command
		if find.Lookup("sort").Document().Lookup("create_time").AsInt64() != -1 || find.Lookup("limit").AsInt64() != 20 {
			mt.Fatal("incorrect default pagination/sort")
		}
	})
	mt.Run("database error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "failed count"}))
		if _, _, err := NewQuestionRepository(mt.Client.Database("catalog")).FilterQuestions(context.Background(), QuestionFilter{}); err == nil {
			mt.Fatal("expected database error")
		}
	})
}

func TestFilterQuestionsSortMappingsAndFindFailure(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	for _, field := range []string{"create_time", "view_count", "thumbs_up_count", "dislike_count"} {
		for _, direction := range []string{"asc", "desc"} {
			mt.Run(field+"/"+direction, func(mt *mtest.T) {
				mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch), mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch))
				if _, _, err := NewQuestionRepository(mt.Client.Database("catalog")).FilterQuestions(context.Background(), QuestionFilter{SortBy: field, SortOrder: direction}); err != nil {
					mt.Fatal(err)
				}
				want := int64(1)
				if direction == "desc" {
					want = -1
				}
				events := mt.GetAllStartedEvents()
				if events[1].Command.Lookup("sort").Document().Lookup(field).AsInt64() != want {
					mt.Fatal("incorrect sort mapping")
				}
			})
		}
	}
	mt.Run("find failure after successful count", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch, bson.D{{Key: "n", Value: int64(2)}}), mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "find failed"}))
		if _, _, err := NewQuestionRepository(mt.Client.Database("catalog")).FilterQuestions(context.Background(), QuestionFilter{}); err == nil {
			mt.Fatal("expected find error")
		}
	})
}

func TestGetQuestionByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("found", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch, bson.D{{Key: "question_id", Value: "q1"}, {Key: "title", Value: "Redis"}, {Key: "content", Value: "完整正文"}, {Key: "status", Value: 2}, {Key: "view_count", Value: 9}}))
		got, err := NewQuestionRepository(mt.Client.Database("catalog")).GetQuestionByID(context.Background(), "q1")
		if err != nil || got.QuestionID != "q1" || got.Content != "完整正文" || got.Status != 2 || got.ViewCount != 9 {
			mt.Fatalf("unexpected detail: %+v, %v", got, err)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 1 || events[0].CommandName != "find" {
			mt.Fatal("detail must only query, not increment views")
		}
		var filter bson.M
		if err := bson.Unmarshal(events[0].Command.Lookup("filter").Document(), &filter); err != nil {
			mt.Fatal(err)
		}
		if !reflect.DeepEqual(filter, bson.M{"question_id": "q1"}) {
			mt.Fatalf("unexpected filter: %v", filter)
		}
	})
	mt.Run("not found", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch))
		if _, err := NewQuestionRepository(mt.Client.Database("catalog")).GetQuestionByID(context.Background(), "missing"); !errors.Is(err, ErrQuestionNotFound) {
			mt.Fatalf("expected not-found error: %v", err)
		}
	})
	mt.Run("query failure", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "failed query"}))
		_, err := NewQuestionRepository(mt.Client.Database("catalog")).GetQuestionByID(context.Background(), "q1")
		if err == nil || errors.Is(err, ErrQuestionNotFound) {
			mt.Fatalf("query failure confused with missing document: %v", err)
		}
	})
}

func TestGetHotQuestionsUsesHeatAndProjection(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("filtered hot query", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch,
			bson.D{{Key: "question_id", Value: "hot"}, {Key: "bank_list", Value: bson.A{"b1"}}, {Key: "title", Value: "热度高"}, {Key: "view_count", Value: 2}},
			bson.D{{Key: "question_id", Value: "views"}, {Key: "bank_list", Value: bson.A{"b2"}}, {Key: "title", Value: "浏览高"}, {Key: "view_count", Value: 100}},
		))
		got, err := NewQuestionRepository(mt.Client.Database("catalog")).GetHotQuestions(context.Background(), "backend", 5)
		if err != nil || len(got) != 2 || got[0].QuestionID != "hot" || got[0].ViewCount != 2 {
			mt.Fatalf("unexpected hot result: %+v, %v", got, err)
		}
		events := mt.GetAllStartedEvents()
		if len(events) != 1 || events[0].CommandName != "find" {
			mt.Fatal("expected independent find")
		}
		command := events[0].Command
		var filter, projection bson.M
		if err := bson.Unmarshal(command.Lookup("filter").Document(), &filter); err != nil {
			mt.Fatal(err)
		}
		if !reflect.DeepEqual(filter, bson.M{"status": int32(1), "job_name": "backend"}) {
			mt.Fatalf("wrong hot filter: %v", filter)
		}
		sort := command.Lookup("sort").Document()
		if sort.Lookup("hot_degree").AsInt64() != -1 || sort.Lookup("question_id").AsInt64() != 1 || sort.Lookup("view_count").Type != 0 {
			mt.Fatal("hot query must sort by heat, not views")
		}
		if command.Lookup("limit").AsInt64() != 5 {
			mt.Fatal("limit not applied")
		}
		if err := bson.Unmarshal(command.Lookup("projection").Document(), &projection); err != nil {
			mt.Fatal(err)
		}
		if !reflect.DeepEqual(projection, bson.M{"_id": int32(0), "question_id": int32(1), "bank_list": int32(1), "title": int32(1), "view_count": int32(1)}) {
			mt.Fatalf("wrong projection: %v", projection)
		}
	})
	mt.Run("empty defaults", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "catalog.question", mtest.FirstBatch))
		got, err := NewQuestionRepository(mt.Client.Database("catalog")).GetHotQuestions(context.Background(), "", 0)
		if err != nil || got == nil || len(got) != 0 {
			mt.Fatalf("empty result: %v %v", got, err)
		}
		command := mt.GetAllStartedEvents()[0].Command
		if command.Lookup("limit").AsInt64() != 10 || lenMustDocument(command.Lookup("filter").Document()) != 1 {
			mt.Fatal("incorrect unfiltered defaults")
		}
	})
	mt.Run("database failure", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 2, Message: "query failed"}))
		if _, err := NewQuestionRepository(mt.Client.Database("catalog")).GetHotQuestions(context.Background(), "", 10); err == nil {
			mt.Fatal("expected error")
		}
	})
}

func lenMustDocument(doc bson.Raw) int { values, _ := doc.Elements(); return len(values) }
