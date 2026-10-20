package service

import (
	"context"
	"errors"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"reflect"
	"strings"
	"testing"
	"time"
)

type catalogStub struct {
	hot      []data.Question
	hotLimit int
	detail   data.Question
	detailID string
	catalog  data.QuestionCatalog
	err      error
	job      string
}

func (s *catalogStub) LoadQuestionCatalog(ctx context.Context, job string) (data.QuestionCatalog, error) {
	s.job = job
	return s.catalog, s.err
}

func TestGetAllQuestionListGroupsAndCounts(t *testing.T) {
	repo := &catalogStub{catalog: data.QuestionCatalog{
		Series: []data.QuestionBankSeries{
			{SeriesID: "s1", SeriesName: "数据库", JobName: "后端开发", Order: 1},
			{SeriesID: "s2", SeriesName: "框架", JobName: "前端开发", Order: 2},
			{SeriesID: "s3", SeriesName: "语言", JobName: "后端开发", Order: 3},
		},
		Banks: []data.QuestionBank{
			{BankID: "b1", SeriesID: "s1", JobName: "后端开发", BankName: "MySQL", BankLogo: "logo", Desc: "简介", Order: 1},
			{BankID: "b2", SeriesID: "s1", JobName: "后端开发", BankName: "Redis", Order: 2},
			{BankID: "b3", SeriesID: "s2", JobName: "前端开发", Order: 1},
		}, Counts: map[string]int64{"b1": 2, "b3": 1},
	}}
	got, err := NewQuestionService(repo).GetAllQuestionList(context.Background(), model.GetAllQuestionListReq{JobName: "后端开发"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.job != "后端开发" {
		t.Fatal("job filter not forwarded")
	}
	if len(got) != 2 || got[0].JobName != "后端开发" || len(got[0].SeriesList) != 2 || got[0].SeriesList[1].SeriesID != "s3" {
		t.Fatalf("incorrect grouping: %+v", got)
	}
	banks := got[0].SeriesList[0].BankList
	if len(banks) != 2 || banks[0].BankID != "b1" || banks[0].Count != 2 || banks[1].Count != 0 || banks[0].BankLogo != "logo" || banks[0].Desc != "简介" {
		t.Fatalf("incorrect banks: %+v", banks)
	}
	if got[0].SeriesList[1].BankList == nil {
		t.Fatal("empty bank list must be []")
	}
}

func TestGetAllQuestionListEmptyAndError(t *testing.T) {
	repo := &catalogStub{}
	got, err := NewQuestionService(repo).GetAllQuestionList(context.Background(), model.GetAllQuestionListReq{})
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("want empty slice, got %v, %v", got, err)
	}
	repo.err = errors.New("database unavailable")
	if _, err := NewQuestionService(repo).GetAllQuestionList(context.Background(), model.GetAllQuestionListReq{}); !errors.Is(err, repo.err) {
		t.Fatalf("lost error: %v", err)
	}
}

// questionListStub 返回持久化文档，用于验证服务层的游客转换。
type questionListStub struct {
	catalogStub
	questions []data.Question
	total     int64
	filter    data.QuestionFilter
}

func (s *questionListStub) FilterQuestions(ctx context.Context, filter data.QuestionFilter) ([]data.Question, int64, error) {
	s.filter = filter
	return s.questions, s.total, s.err
}
func (s *catalogStub) FilterQuestions(context.Context, data.QuestionFilter) ([]data.Question, int64, error) {
	return nil, 0, s.err
}

func TestListQuestionsGuestContentAndFields(t *testing.T) {
	when := time.Date(2026, 10, 19, 12, 30, 0, 0, time.UTC)
	repo := &questionListStub{questions: []data.Question{{
		QuestionID: "q1", BankList: []string{"b1"}, Title: "Redis", Content: strings.Repeat("中", 150) + "🙂尾",
		Difficulty: 2, Tags: []string{"Redis"}, Status: 1, VIP: true, HotDegree: 12,
		ViewCount: 3, ThumbsUpCount: 4, DislikeCount: 5, Order: 9, CreateTime: when, UpdateTime: when,
	}}, total: 40}
	req := model.ListQuestionReq{BankID: "b1", Keyword: "Redis", Difficulty: 2, Tags: []string{"Redis"}, JobName: "backend", UserTag: 3, Page: 2, PageSize: 20, SortBy: "view_count", SortOrder: "asc"}
	got, err := NewQuestionService(repo).ListQuestions(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 40 || len(got.List) != 1 {
		t.Fatalf("wrong pagination result: %+v", got)
	}
	q := got.List[0]
	if q.Content != strings.Repeat("中", 150) || q.UserTag != 0 || q.UserLiked {
		t.Fatalf("incorrect guest projection: %+v", q)
	}
	if q.QuestionID != "q1" || q.VIP != true || q.Order != 9 || q.ViewCount != 3 || q.ThumbsUpCount != 4 || q.DislikeCount != 5 || q.HotDegree != 12 || q.CreateTime != "2026-10-19 12:30:00" || q.UpdateTime != q.CreateTime {
		t.Fatalf("lost document fields: %+v", q)
	}
	if repo.questions[0].Content != strings.Repeat("中", 150)+"🙂尾" {
		t.Fatal("service mutated stored document")
	}
	if repo.filter.BankID != "b1" || repo.filter.Keyword != "Redis" || repo.filter.JobName != "backend" || repo.filter.Page != 2 || repo.filter.PageSize != 20 || repo.filter.SortBy != "view_count" || repo.filter.SortOrder != "asc" || repo.filter.Difficulty != 2 || len(repo.filter.Tags) != 1 {
		t.Fatalf("lost filters: %+v", repo.filter)
	}
}

func TestListQuestionsEmptyShortContentAndError(t *testing.T) {
	repo := &questionListStub{}
	svc := NewQuestionService(repo)
	got, err := svc.ListQuestions(context.Background(), model.ListQuestionReq{Page: 1, PageSize: 20, SortBy: "create_time", SortOrder: "desc"})
	if err != nil || got.List == nil || len(got.List) != 0 {
		t.Fatalf("empty response: %+v, %v", got, err)
	}
	repo.questions = []data.Question{{Content: "短内容"}}
	got, err = svc.ListQuestions(context.Background(), model.ListQuestionReq{})
	if err != nil || got.List[0].Content != "短内容" || got.List[0].BankList == nil || got.List[0].Tags == nil {
		t.Fatalf("short content response: %+v, %v", got, err)
	}
	repo.err = errors.New("query failed")
	if _, err := svc.ListQuestions(context.Background(), model.ListQuestionReq{}); !errors.Is(err, repo.err) {
		t.Fatalf("lost query error: %v", err)
	}
}

func TestListQuestionsMetaReusesFiltersAndReturnsMinimalFields(t *testing.T) {
	repo := &questionListStub{questions: []data.Question{{QuestionID: "q1", Title: "Redis", Content: "不应返回的内容", Status: 1}, {QuestionID: "q2", Title: "MySQL"}}, total: 31}
	svc := NewQuestionService(repo)
	req := model.ListQuestionMetaReq{BankID: "b1", Keyword: "Redis", Tags: []string{"DB", "Redis"}, Difficulty: 2, JobName: "backend", UserTag: 3, SortBy: "view_count", SortOrder: "asc", Page: 2, PageSize: 10}
	normal, err := svc.ListQuestions(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	listFilter := repo.filter
	meta, err := svc.ListQuestionsMeta(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(listFilter, repo.filter) {
		t.Fatalf("list and meta filters differ: %+v / %+v", listFilter, repo.filter)
	}
	if meta.Code != 0 || meta.Msg != "success" || meta.Data.Total != normal.Total || len(meta.Data.List) != 2 || meta.Data.List[0].QuestionID != "q1" || meta.Data.List[0].Title != "Redis" || meta.Data.List[1].QuestionID != "q2" {
		t.Fatalf("incorrect meta response: %+v", meta)
	}
	repo.questions = nil
	empty, err := svc.ListQuestionsMeta(context.Background(), req)
	if err != nil || empty.Data.List == nil {
		t.Fatalf("empty list must be []: %+v, %v", empty, err)
	}
	repo.err = errors.New("repository failed")
	if _, err := svc.ListQuestionsMeta(context.Background(), req); !errors.Is(err, repo.err) {
		t.Fatalf("lost repository error: %v", err)
	}
}

func (s *catalogStub) GetQuestionByID(ctx context.Context, id string) (data.Question, error) {
	s.detailID = id
	return s.detail, s.err
}

func TestGetQuestionDetailGuestContent(t *testing.T) {
	content := strings.Repeat("中文🙂", 100)
	repo := &catalogStub{detail: data.Question{QuestionID: "q1", Title: "Redis", Content: content, Status: 1, Order: 7, ViewCount: 8, CreateTime: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)}}
	got, err := NewQuestionService(repo).GetQuestionDetail(context.Background(), model.GetQuestionDetailReq{QuestionID: "q1"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.detailID != "q1" || got.QuestionID != "q1" || got.Content != string([]rune(content)[:150]) || got.UserTag != 0 || got.UserLiked || got.Order != 7 || got.ViewCount != 8 || got.CreateTime != "2026-10-19 12:00:00" || got.BankList == nil || got.Tags == nil {
		t.Fatalf("incorrect detail response: %+v", got)
	}
	for _, failure := range []error{data.ErrQuestionNotFound, errors.New("database failed")} {
		repo.err = failure
		if _, err := NewQuestionService(repo).GetQuestionDetail(context.Background(), model.GetQuestionDetailReq{QuestionID: "q1"}); !errors.Is(err, failure) {
			t.Fatalf("lost error classification: %v", err)
		}
	}
}

func (*catalogStub) GetQuestionUserState(context.Context, string, string) (data.QuestionUserState, error) {
	return data.QuestionUserState{}, nil
}

type detailStateRepo struct {
	catalogStub
	state              data.QuestionUserState
	stateErr           error
	stateCalls         int
	userID, questionID string
}

func (r *detailStateRepo) GetQuestionUserState(_ context.Context, userID, questionID string) (data.QuestionUserState, error) {
	r.stateCalls++
	r.userID = userID
	r.questionID = questionID
	return r.state, r.stateErr
}
func TestQuestionDetailVisibilityAndPersonalization(t *testing.T) {
	for _, userID := range []string{"", "user1"} {
		for _, content := range []string{"", "短🙂", strings.Repeat("中", 149) + "🙂尾"} {
			r := &detailStateRepo{catalogStub: catalogStub{detail: data.Question{QuestionID: "q1", Content: content, AnalysisContent: "完整解析"}}, state: data.QuestionUserState{UserLiked: true, UserTag: 3}}
			got, err := NewQuestionService(r).GetQuestionDetail(context.Background(), model.GetQuestionDetailReq{QuestionID: "q1", UserID: userID})
			if err != nil {
				t.Fatal(err)
			}
			if userID == "" {
				want := []rune(content)
				if len(want) > 150 {
					want = want[:150]
				}
				if got.Content != string(want) || got.AnalysisContent != "" || got.UserLiked || got.UserTag != 0 || r.stateCalls != 0 {
					t.Fatalf("guest leaked data %+v", got)
				}
			} else if got.Content != content || got.AnalysisContent != "完整解析" || !got.UserLiked || got.UserTag != 3 || r.userID != "user1" || r.questionID != "q1" {
				t.Fatalf("incorrect logged in response %+v", got)
			}
		}
	}
	r := &detailStateRepo{catalogStub: catalogStub{detail: data.Question{QuestionID: "q1"}}}
	got, err := NewQuestionService(r).GetQuestionDetail(context.Background(), model.GetQuestionDetailReq{QuestionID: "q1", UserID: "user1"})
	if err != nil || got.UserLiked || got.UserTag != 0 {
		t.Fatalf("missing state %+v %v", got, err)
	}
	failure := errors.New("state lookup failed")
	r.stateErr = failure
	if _, err := NewQuestionService(r).GetQuestionDetail(context.Background(), model.GetQuestionDetailReq{QuestionID: "q1", UserID: "user1"}); !errors.Is(err, failure) {
		t.Fatalf("state error lost %v", err)
	}
	r.err = data.ErrQuestionNotFound
	r.stateCalls = 0
	if _, err := NewQuestionService(r).GetQuestionDetail(context.Background(), model.GetQuestionDetailReq{QuestionID: "missing", UserID: "user1"}); !errors.Is(err, data.ErrQuestionNotFound) || r.stateCalls != 0 {
		t.Fatal("state queried for missing question")
	}
}

func (s *catalogStub) GetHotQuestions(ctx context.Context, job string, limit int) ([]data.Question, error) {
	s.job = job
	s.hotLimit = limit
	return s.hot, s.err
}

func TestGetHotQuestionsResponseAndError(t *testing.T) {
	repo := &catalogStub{hot: []data.Question{{QuestionID: "hot", Title: "Redis", BankList: []string{"b1"}, ViewCount: 2, Content: "不应返回"}, {QuestionID: "views", Title: "MySQL", ViewCount: 100}}}
	svc := NewQuestionService(repo)
	got, err := svc.GetHotQuestions(context.Background(), model.GetHotQuestionsReq{JobName: "backend", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if repo.job != "backend" || repo.hotLimit != 5 || got.Code != 0 || got.Msg != "success" || len(got.Data.List) != 2 || got.Data.List[0].QuestionID != "hot" || got.Data.List[0].BankList[0] != "b1" || got.Data.List[0].ViewCount != 2 || got.Data.List[1].BankList == nil {
		t.Fatalf("incorrect hot response: %+v", got)
	}
	repo.hot = nil
	got, err = svc.GetHotQuestions(context.Background(), model.GetHotQuestionsReq{})
	if err != nil || got.Data.List == nil || len(got.Data.List) != 0 {
		t.Fatalf("empty list: %+v %v", got, err)
	}
	repo.err = errors.New("database failed")
	if _, err := svc.GetHotQuestions(context.Background(), model.GetHotQuestionsReq{}); !errors.Is(err, repo.err) {
		t.Fatalf("lost error: %v", err)
	}
}
