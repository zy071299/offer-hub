package question

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"offer-hub/backend/src/service"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"offer-hub/backend/src/model"
)

type listStub struct {
	hotReq      model.GetHotQuestionsReq
	hotCalls    int
	detailID    string
	detailCalls int
	listReq     model.ListQuestionReq
	listCalls   int
	job         string
	err         error
	canceled    bool
}

func (s *listStub) GetAllQuestionList(ctx context.Context, req model.GetAllQuestionListReq) ([]model.GetAllQuestionListData, error) {
	s.job = req.JobName
	s.canceled = ctx.Err() != nil
	return []model.GetAllQuestionListData{}, s.err
}
func TestGetAllQuestionListQueryAndResponse(t *testing.T) {
	stub := &listStub{}
	router := gin.New()
	router.GET("/api/v1/question/all/list", NewController(stub).GetAllQuestionList)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/question/all/list?job_name=backend", nil)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	router.ServeHTTP(w, req.WithContext(ctx))
	if stub.job != "backend" || !stub.canceled {
		t.Fatal("query or request context not forwarded")
	}
	var got struct {
		Code int                            `json:"code"`
		Msg  string                         `json:"msg"`
		Data []model.GetAllQuestionListData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || got.Code != 0 || got.Msg != "success" || got.Data == nil {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body)
	}
	stub.err = errors.New("private database details")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/question/all/list", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":500`) || !strings.Contains(w.Body.String(), `"data":null`) || strings.Contains(w.Body.String(), "private database") {
		t.Fatalf("unexpected error response: %s", w.Body)
	}
}

func TestGetAllQuestionListRejectsMalformedQuery(t *testing.T) {
	stub := &listStub{}
	router := gin.New()
	router.GET("/", NewController(stub).GetAllQuestionList)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/?job_name=backend;bad", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":400`) {
		t.Fatalf("unexpected binding response: %s", w.Body)
	}
}

// ListQuestions 在同一控制器中保持 HTTP 请求处理与业务服务分离。
func (s *listStub) ListQuestions(ctx context.Context, req model.ListQuestionReq) (model.ListQuestionData, error) {
	s.listReq = req
	s.listCalls++
	return model.ListQuestionData{Total: 2, List: []model.OneQuestion{{QuestionID: "q1", Content: "内容"}}}, s.err
}

func TestListQuestionsDefaultsBindingAndResponse(t *testing.T) {
	stub := &listStub{}
	router := gin.New()
	router.GET("/list", NewController(stub).ListQuestions)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/list", nil))
	if stub.listReq.Page != 1 || stub.listReq.PageSize != 20 || stub.listReq.SortBy != "create_time" || stub.listReq.SortOrder != "desc" {
		t.Fatalf("incorrect defaults: %+v", stub.listReq)
	}
	var got model.ListQuestionResp
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || got.Code != 0 || got.Msg != "success" || got.Data.Total != 2 || got.Data.List[0].QuestionID != "q1" {
		t.Fatalf("unexpected response: %s", w.Body)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/list?bank_id=b1&keyword=Go&difficulty=2&tags=Go&tags=DB&job_name=backend&user_tag=3&page=2&page_size=10&sort_by=view_count&sort_order=asc", nil))
	req := stub.listReq
	if req.BankID != "b1" || req.Keyword != "Go" || req.Difficulty != 2 || len(req.Tags) != 2 || req.JobName != "backend" || req.Page != 2 || req.PageSize != 10 || req.SortBy != "view_count" || req.SortOrder != "asc" {
		t.Fatalf("binding lost parameters: %+v", req)
	}
	stub.err = errors.New("private database details")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/list", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":500`) || !strings.Contains(w.Body.String(), `"data":null`) || strings.Contains(w.Body.String(), "private database") {
		t.Fatalf("unexpected error: %s", w.Body)
	}
}

func TestListQuestionsInvalidParameters(t *testing.T) {
	for _, query := range []string{"page=0", "page=-1", "page=x", "page_size=0", "page_size=-2", "difficulty=4", "difficulty=-1", "sort_by=content", "sort_order=up", "page=9223372036854775807&page_size=20", "keyword=bad;query"} {
		t.Run(query, func(t *testing.T) {
			stub := &listStub{}
			router := gin.New()
			router.GET("/list", NewController(stub).ListQuestions)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/list?"+query, nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":400`) || stub.listCalls != 0 {
				t.Fatalf("invalid query reached service: %s", w.Body)
			}
		})
	}
}

func (s *listStub) ListQuestionsMeta(ctx context.Context, req model.ListQuestionMetaReq) (model.ListQuestionMetaResp, error) {
	s.listReq = req
	s.listCalls++
	s.canceled = ctx.Err() != nil
	return model.ListQuestionMetaResp{Code: 0, Msg: "success", Data: model.ListQuestionMetaData{Total: 2, List: []model.QuestionMetaInfo{{QuestionID: "q1", Title: "Redis"}}}}, s.err
}

func TestListQuestionsMetaMatchesListBindingAndMinimalResponse(t *testing.T) {
	stub := &listStub{}
	router := gin.New()
	ctrl := NewController(stub)
	router.GET("/list", ctrl.ListQuestions)
	router.GET("/meta", ctrl.ListQuestionsMeta)
	for _, query := range []string{"", "?bank_id=b1&keyword=Go&tags=Go&tags=DB&difficulty=2&job_name=backend&user_tag=3&page=2&page_size=10&sort_by=dislike_count&sort_order=asc"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/list"+query, nil))
		listReq := stub.listReq
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/meta"+query, nil))
		if !reflect.DeepEqual(stub.listReq, listReq) {
			t.Fatalf("inconsistent binding: %+v / %+v", stub.listReq, listReq)
		}
		var body struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Total int64            `json:"total"`
				List  []map[string]any `json:"list"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || body.Code != 0 || body.Msg != "success" || body.Data.Total != 2 || len(body.Data.List) != 1 {
			t.Fatalf("unexpected meta response: %s", w.Body)
		}
		item := body.Data.List[0]
		if len(item) != 2 || item["question_id"] != "q1" || item["title"] != "Redis" {
			t.Fatalf("unexpected meta fields: %v", item)
		}
	}
	stub.err = errors.New("private failure details")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/meta", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":500`) || !strings.Contains(w.Body.String(), `"data":null`) || strings.Contains(w.Body.String(), "private failure") {
		t.Fatalf("unexpected meta error: %s", w.Body)
	}
}

func TestListQuestionsMetaRejectsInvalidParameters(t *testing.T) {
	for _, query := range []string{"page=0", "page=-1", "page=x", "page_size=0", "page_size=-2", "difficulty=4", "difficulty=-1", "sort_by=content", "sort_order=up", "page=9223372036854775807&page_size=20", "keyword=bad;query"} {
		t.Run(query, func(t *testing.T) {
			stub := &listStub{}
			router := gin.New()
			router.GET("/meta", NewController(stub).ListQuestionsMeta)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/meta?"+query, nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":400`) || stub.listCalls != 0 {
				t.Fatalf("invalid meta query reached service: %s", w.Body)
			}
		})
	}
}

func (s *listStub) GetQuestionDetail(ctx context.Context, req model.GetQuestionDetailReq) (model.OneQuestion, error) {
	s.detailID = req.QuestionID
	s.detailCalls++
	return model.OneQuestion{QuestionID: req.QuestionID, Title: "Redis", Content: strings.Repeat("中", 200)}, s.err
}

func TestGetQuestionDetailBindingAndResponses(t *testing.T) {
	stub := &listStub{}
	router := gin.New()
	router.GET("/detail", NewController(stub).GetQuestionDetail)
	for _, query := range []string{"", "?question_id=", "?question_id=%20%20", "?question_id=bad;query"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/detail"+query, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":400`) || stub.detailCalls != 0 {
			t.Fatalf("invalid detail query reached service: %s", w.Body)
		}
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/detail?question_id=q1", nil))
	var got model.GetQuestionDetailResp
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || got.Code != 0 || got.Msg != "success" || got.Data.QuestionID != "q1" || got.Data.Content != strings.Repeat("中", 200) || stub.detailID != "q1" {
		t.Fatalf("unexpected detail: %s", w.Body)
	}
	for _, tc := range []struct {
		err  error
		code int
	}{{fmt.Errorf("wrapped: %w", service.ErrQuestionNotFound), 404}, {errors.New("private db failure"), 500}} {
		stub.err = tc.err
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/detail?question_id=q1", nil))
		var body struct {
			Code int `json:"code"`
			Data any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || body.Code != tc.code || body.Data != nil || strings.Contains(w.Body.String(), "private db") {
			t.Fatalf("unexpected error response: %s", w.Body)
		}
	}
}

func (s *listStub) GetHotQuestions(ctx context.Context, req model.GetHotQuestionsReq) (model.GetHotQuestionsResp, error) {
	s.hotReq = req
	s.hotCalls++
	return model.GetHotQuestionsResp{Code: 0, Msg: "success", Data: model.GetHotQuestionListData{List: []model.HotQuestionInfo{{QuestionID: "q1", BankList: []string{"b1"}, Title: "Redis", ViewCount: 3}}}}, s.err
}

func TestGetHotQuestionsParametersAndResponse(t *testing.T) {
	stub := &listStub{}
	router := gin.New()
	router.GET("/hot", NewController(stub).GetHotQuestions)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/hot", nil))
	if stub.hotReq.Limit != 10 || stub.hotReq.JobName != "" {
		t.Fatalf("incorrect defaults: %+v", stub.hotReq)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/hot?job_name=backend&limit=5", nil))
	if stub.hotReq.Limit != 5 || stub.hotReq.JobName != "backend" {
		t.Fatalf("lost parameters: %+v", stub.hotReq)
	}
	var body struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			List []map[string]any `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || body.Code != 0 || body.Msg != "success" || len(body.Data.List) != 1 || len(body.Data.List[0]) != 4 || body.Data.List[0]["question_id"] != "q1" || body.Data.List[0]["title"] != "Redis" || body.Data.List[0]["view_count"] != float64(3) {
		t.Fatalf("incorrect hot response: %s", w.Body)
	}
	stub.err = errors.New("private db failure")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/hot", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":500`) || !strings.Contains(w.Body.String(), `"data":null`) || strings.Contains(w.Body.String(), "private db") {
		t.Fatalf("unexpected error: %s", w.Body)
	}
}

func TestGetHotQuestionsRejectsInvalidLimit(t *testing.T) {
	for _, query := range []string{"limit=0", "limit=-1", "limit=x", "limit=", "limit=9223372036854775808", "job_name=bad;query"} {
		t.Run(query, func(t *testing.T) {
			stub := &listStub{}
			router := gin.New()
			router.GET("/hot", NewController(stub).GetHotQuestions)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/hot?"+query, nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":400`) || stub.hotCalls != 0 {
				t.Fatalf("invalid hot request reached service: %s", w.Body)
			}
		})
	}
}
