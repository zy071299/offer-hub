package model

// ListQuestionReq 用于题目列表及题目元数据列表查询。
// 控制器设置分页和排序默认值；游客过滤规则由业务层处理。
type ListQuestionReq struct {
	BankID     string   `json:"bank_id" form:"bank_id"`
	Keyword    string   `json:"keyword" form:"keyword"`
	Difficulty int      `json:"difficulty" form:"difficulty"`
	Tags       []string `json:"tags" form:"tags"`
	JobName    string   `json:"job_name" form:"job_name"`
	UserTag    int      `json:"user_tag" form:"user_tag"`
	SortBy     string   `json:"sort_by" form:"sort_by"`
	SortOrder  string   `json:"sort_order" form:"sort_order"`
	Page       int      `json:"page" form:"page"`
	PageSize   int      `json:"page_size" form:"page_size"`
}

// OneQuestion 是列表和详情共用的题目信息。
// 时间格式为 YYYY-MM-DD HH:mm:ss。
type OneQuestion struct {
	QuestionID      string   `json:"question_id"`
	BankList        []string `json:"bank_list"`
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	AnalysisContent string   `json:"analysis_content"`
	Difficulty      int      `json:"difficulty"`
	Tags            []string `json:"tags"`
	Status          int      `json:"status"`
	VIP             bool     `json:"vip"`
	HotDegree       int      `json:"hot_degree"`
	ViewCount       int      `json:"view_count"`
	ThumbsUpCount   int      `json:"thumbs_up_count"`
	DislikeCount    int      `json:"dislike_count"`
	Order           int64    `json:"order"`
	UserTag         int      `json:"user_tag"`
	UserLiked       bool     `json:"user_liked"`
	CreateTime      string   `json:"create_time"`
	UpdateTime      string   `json:"update_time"`
}

type ListQuestionData struct {
	Total int64         `json:"total"`
	List  []OneQuestion `json:"list"`
}

type ListQuestionResp struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data ListQuestionData `json:"data"`
}

type GetQuestionDetailReq struct {
	QuestionID string `json:"question_id" form:"question_id"`
	UserID     string `json:"-" form:"-"`
}

type GetQuestionDetailResp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data OneQuestion `json:"data"`
}

type HotQuestionInfo struct {
	QuestionID string   `json:"question_id"`
	BankList   []string `json:"bank_list"`
	Title      string   `json:"title"`
	ViewCount  int      `json:"view_count"`
}

type GetHotQuestionListReq struct {
	Limit   int    `json:"limit" form:"limit"`
	JobName string `json:"job_name" form:"job_name"`
}

type GetHotQuestionListData struct {
	List []HotQuestionInfo `json:"list"`
}

type GetHotQuestionListResp struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data GetHotQuestionListData `json:"data"`
}

type GetQuestionBankSeriesReq struct {
	JobName string `json:"job_name" form:"job_name"`
}

// GetQuestionBankSeriesData 是一个职位方向的题库分组。
// 全量列表响应的 data 是此结构体的数组。
type GetQuestionBankSeriesData struct {
	JobName    string               `json:"job_name"`
	SeriesList []QuestionBankSeries `json:"series_list"`
}

type QuestionBankSeries struct {
	SeriesID   string             `json:"series_id"`
	SeriesName string             `json:"series_name"`
	Order      int64              `json:"order"`
	BankList   []QuestionBankInfo `json:"bank_list"`
}

type QuestionBankInfo struct {
	BankID   string `json:"bank_id"`
	BankName string `json:"bank_name"`
	BankLogo string `json:"bank_logo"`
	Desc     string `json:"desc"`
	Count    int    `json:"count"`
	Order    int64  `json:"order"`
}

type GetQuestionBankSeriesResp struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data []GetQuestionBankSeriesData `json:"data"`
}

type QuestionMetaInfo struct {
	QuestionID string `json:"question_id"`
	Title      string `json:"title"`
}

type ListQuestionMetaReq = ListQuestionReq

type ListQuestionMetaData struct {
	Total int64              `json:"total"`
	List  []QuestionMetaInfo `json:"list"`
}

type ListQuestionMetaResp struct {
	Code int                  `json:"code"`
	Msg  string               `json:"msg"`
	Data ListQuestionMetaData `json:"data"`
}

type TagQuestionReq struct {
	QuestionID string `json:"question_id" form:"question_id"`
	Tag        int    `json:"tag" form:"tag"`
}

// TagQuestionResp 的 Data 保持 nil 时序列化为文档规定的 null。
type TagQuestionResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// GetAllQuestionListReq 查询题库全量列表，按职位方向过滤。
type GetAllQuestionListReq = GetQuestionBankSeriesReq

// GetAllQuestionListData 为一个职位方向及其系列、题库列表。
type GetAllQuestionListData = GetQuestionBankSeriesData

type GetAllQuestionListResp struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data []GetAllQuestionListData `json:"data"`
}

// GetHotQuestionsReq 和 GetHotQuestionsResp 复用已有热门题目 DTO。
type GetHotQuestionsReq = GetHotQuestionListReq

type GetHotQuestionsResp = GetHotQuestionListResp
