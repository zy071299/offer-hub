package service

import (
	"context"
	"fmt"
	"offer-hub/backend/src/data"
	"offer-hub/backend/src/model"
	"time"
)

type QuestionRepository interface {
	LoadQuestionCatalog(context.Context, string) (data.QuestionCatalog, error)
	FilterQuestions(context.Context, data.QuestionFilter) ([]data.Question, int64, error)
	GetQuestionByID(context.Context, string) (data.Question, error)
	GetHotQuestions(context.Context, string, int) ([]data.Question, error)
}

type QuestionService struct{ repository QuestionRepository }

func NewQuestionService(repository QuestionRepository) *QuestionService {
	if repository == nil {
		repository = data.QuestionRepository{}
	}
	return &QuestionService{repository: repository}
}

// GetAllQuestionList 使用全局数据库连接供 HTTP 控制器调用。
func GetAllQuestionList(ctx context.Context, req model.GetAllQuestionListReq) ([]model.GetAllQuestionListData, error) {
	return NewQuestionService(data.QuestionRepository{}).GetAllQuestionList(ctx, req)
}

func (s *QuestionService) GetAllQuestionList(ctx context.Context, req model.GetAllQuestionListReq) ([]model.GetAllQuestionListData, error) {
	catalog, err := s.repository.LoadQuestionCatalog(ctx, req.JobName)
	if err != nil {
		return nil, fmt.Errorf("load question catalog: %w", err)
	}
	result := make([]model.GetAllQuestionListData, 0)
	banksBySeries := make(map[string][]model.QuestionBankInfo)
	// 数据层已按 order 升序，按遍历顺序追加，避免 map 迭代打乱顺序。
	for _, bank := range catalog.Banks {
		banksBySeries[bank.SeriesID] = append(banksBySeries[bank.SeriesID], model.QuestionBankInfo{
			BankID: bank.BankID, BankName: bank.BankName, BankLogo: bank.BankLogo, Desc: bank.Desc,
			Count: int(catalog.Counts[bank.BankID]), Order: bank.Order,
		})
	}
	jobIndex := make(map[string]int)
	for _, series := range catalog.Series {
		index, ok := jobIndex[series.JobName]
		if !ok {
			index = len(result)
			jobIndex[series.JobName] = index
			result = append(result, model.GetAllQuestionListData{JobName: series.JobName, SeriesList: []model.QuestionBankSeries{}})
		}
		banks := banksBySeries[series.SeriesID]
		if banks == nil {
			banks = []model.QuestionBankInfo{}
		}
		result[index].SeriesList = append(result[index].SeriesList, model.QuestionBankSeries{
			SeriesID: series.SeriesID, SeriesName: series.SeriesName, Order: series.Order, BankList: banks,
		})
	}
	return result, nil
}

// ListQuestions 返回游客题目列表；忽略掌握状态过滤，用户动态字段保持零值。
func ListQuestions(ctx context.Context, req model.ListQuestionReq) (model.ListQuestionData, error) {
	return NewQuestionService(nil).ListQuestions(ctx, req)
}

func (s *QuestionService) ListQuestions(ctx context.Context, req model.ListQuestionReq) (model.ListQuestionData, error) {
	questions, total, err := s.filterQuestions(ctx, req)
	if err != nil {
		return model.ListQuestionData{}, fmt.Errorf("list questions: %w", err)
	}
	result := model.ListQuestionData{Total: total, List: make([]model.OneQuestion, 0, len(questions))}
	for _, question := range questions {
		item := toOneQuestion(question)
		content := []rune(item.Content)
		if len(content) > 150 {
			item.Content = string(content[:150])
		}
		result.List = append(result.List, item)
	}
	return result, nil
}

func formatQuestionTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

// filterQuestions 是列表和元数据共用的过滤入口。
// 当前均按游客规则忽略 user_tag，后续用户过滤应统一在此扩展。
func (s *QuestionService) filterQuestions(ctx context.Context, req model.ListQuestionReq) ([]data.Question, int64, error) {
	return s.repository.FilterQuestions(ctx, data.QuestionFilter{
		BankID: req.BankID, Keyword: req.Keyword, Difficulty: req.Difficulty, Tags: req.Tags, JobName: req.JobName,
		SortBy: req.SortBy, SortOrder: req.SortOrder, Page: req.Page, PageSize: req.PageSize,
	})
}

func ListQuestionsMeta(ctx context.Context, req model.ListQuestionMetaReq) (model.ListQuestionMetaResp, error) {
	return NewQuestionService(nil).ListQuestionsMeta(ctx, req)
}

// ListQuestionsMeta 复用列表查询，只组装 ID、标题，不执行正文展示转换。
func (s *QuestionService) ListQuestionsMeta(ctx context.Context, req model.ListQuestionMetaReq) (model.ListQuestionMetaResp, error) {
	questions, total, err := s.filterQuestions(ctx, req)
	if err != nil {
		return model.ListQuestionMetaResp{}, fmt.Errorf("list questions meta: %w", err)
	}
	result := model.ListQuestionMetaResp{Code: 0, Msg: "success", Data: model.ListQuestionMetaData{Total: total, List: make([]model.QuestionMetaInfo, 0, len(questions))}}
	for _, question := range questions {
		result.Data.List = append(result.Data.List, model.QuestionMetaInfo{QuestionID: question.QuestionID, Title: question.Title})
	}
	return result, nil
}

// ErrQuestionNotFound 暴露服务层可识别的题目不存在错误。
var ErrQuestionNotFound = data.ErrQuestionNotFound

func GetQuestionDetail(ctx context.Context, req model.GetQuestionDetailReq) (model.OneQuestion, error) {
	return NewQuestionService(nil).GetQuestionDetail(ctx, req)
}

// GetQuestionDetail 当前返回完整内容，用户个性化字段与列表一致为零值。
func (s *QuestionService) GetQuestionDetail(ctx context.Context, req model.GetQuestionDetailReq) (model.OneQuestion, error) {
	question, err := s.repository.GetQuestionByID(ctx, req.QuestionID)
	if err != nil {
		return model.OneQuestion{}, fmt.Errorf("get question detail: %w", err)
	}
	return toOneQuestion(question), nil
}

// toOneQuestion 只转换字段；各接口的内容展示策略由各自业务方法处理。
func toOneQuestion(question data.Question) model.OneQuestion {
	banks := question.BankList
	if banks == nil {
		banks = []string{}
	}
	tags := question.Tags
	if tags == nil {
		tags = []string{}
	}
	return model.OneQuestion{
		QuestionID: question.QuestionID, BankList: banks, Title: question.Title, Content: question.Content,
		Difficulty: question.Difficulty, Tags: tags, Status: question.Status, VIP: question.VIP,
		HotDegree: question.HotDegree, ViewCount: question.ViewCount, ThumbsUpCount: question.ThumbsUpCount,
		DislikeCount: question.DislikeCount, Order: question.Order, UserTag: 0, UserLiked: false,
		CreateTime: formatQuestionTime(question.CreateTime), UpdateTime: formatQuestionTime(question.UpdateTime),
	}
}

func GetHotQuestions(ctx context.Context, req model.GetHotQuestionsReq) (model.GetHotQuestionsResp, error) {
	return NewQuestionService(nil).GetHotQuestions(ctx, req)
}

// GetHotQuestions 保留数据层的热度排序，只组装接口要求的四个字段。
func (s *QuestionService) GetHotQuestions(ctx context.Context, req model.GetHotQuestionsReq) (model.GetHotQuestionsResp, error) {
	questions, err := s.repository.GetHotQuestions(ctx, req.JobName, req.Limit)
	if err != nil {
		return model.GetHotQuestionsResp{}, fmt.Errorf("get hot questions: %w", err)
	}
	result := model.GetHotQuestionsResp{Code: 0, Msg: "success", Data: model.GetHotQuestionListData{List: make([]model.HotQuestionInfo, 0, len(questions))}}
	for _, question := range questions {
		banks := question.BankList
		if banks == nil {
			banks = []string{}
		}
		result.Data.List = append(result.Data.List, model.HotQuestionInfo{QuestionID: question.QuestionID, BankList: banks, Title: question.Title, ViewCount: question.ViewCount})
	}
	return result, nil
}
