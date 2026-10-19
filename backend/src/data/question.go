// MongoDB 查询与持久化结构，与 HTTP DTO 分离。
package data

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"offer-hub/backend/src/config"
)

const QuestionStatusNormal = 1

// QuestionBankSeries 对应 question_bank_series 集合。
type QuestionBankSeries struct {
	ID         primitive.ObjectID `bson:"_id"`
	SeriesID   string             `bson:"series_id"`
	SeriesName string             `bson:"series_name"`
	JobName    string             `bson:"job_name"`
	Order      int64              `bson:"order"`
	CreateTime time.Time          `bson:"create_time"`
	UpdateTime time.Time          `bson:"update_time"`
}

// QuestionBank 对应 question_bank 集合；数量通过 question 集合动态统计。
type QuestionBank struct {
	ID         primitive.ObjectID `bson:"_id"`
	BankID     string             `bson:"bank_id"`
	SeriesID   string             `bson:"series_id"`
	BankName   string             `bson:"bank_name"`
	BankLogo   string             `bson:"bank_logo"`
	Desc       string             `bson:"desc"`
	JobName    string             `bson:"job_name"`
	Order      int64              `bson:"order"`
	CreateTime time.Time          `bson:"create_time"`
	UpdateTime time.Time          `bson:"update_time"`
}

type QuestionCatalog struct {
	Series []QuestionBankSeries
	Banks  []QuestionBank
	Counts map[string]int64
}

// QuestionRepository 的零值使用已初始化的全局 MongoDB。
// 也可通过 NewQuestionRepository 指定数据库。
type QuestionRepository struct{ db *mongo.Database }

func NewQuestionRepository(db *mongo.Database) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// LoadQuestionCatalog 查询系列、题库并一次性统计全部返回题库的正常题目数。
func (r QuestionRepository) LoadQuestionCatalog(ctx context.Context, jobName string) (QuestionCatalog, error) {
	result := QuestionCatalog{Series: []QuestionBankSeries{}, Banks: []QuestionBank{}, Counts: map[string]int64{}}
	db, err := r.database()
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	filter := bson.D{}
	if jobName != "" {
		filter = append(filter, bson.E{Key: "job_name", Value: jobName})
	}
	cursor, err := db.Collection("question_bank_series").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "series_id", Value: 1}}))
	if err != nil {
		return result, fmt.Errorf("find question_bank_series: %w", err)
	}
	if err = cursor.All(ctx, &result.Series); err != nil {
		return result, fmt.Errorf("decode question_bank_series: %w", err)
	}
	if len(result.Series) == 0 {
		return result, nil
	}
	seriesIDs := make([]string, 0, len(result.Series))
	for _, series := range result.Series {
		seriesIDs = append(seriesIDs, series.SeriesID)
	}
	bankFilter := bson.D{{Key: "series_id", Value: bson.D{{Key: "$in", Value: seriesIDs}}}}
	if jobName != "" {
		bankFilter = append(bankFilter, bson.E{Key: "job_name", Value: jobName})
	}
	cursor, err = db.Collection("question_bank").Find(ctx, bankFilter, options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "bank_id", Value: 1}}))
	if err != nil {
		return result, fmt.Errorf("find question_bank: %w", err)
	}
	if err = cursor.All(ctx, &result.Banks); err != nil {
		return result, fmt.Errorf("decode question_bank: %w", err)
	}
	if len(result.Banks) == 0 {
		return result, nil
	}
	bankIDs := make([]string, 0, len(result.Banks))
	for _, bank := range result.Banks {
		bankIDs = append(bankIDs, bank.BankID)
	}
	bankMatch := bson.D{{Key: "$in", Value: bankIDs}}
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{{Key: "status", Value: QuestionStatusNormal}, {Key: "bank_list", Value: bankMatch}}}},
		bson.D{{Key: "$unwind", Value: "$bank_list"}},
		bson.D{{Key: "$match", Value: bson.D{{Key: "bank_list", Value: bankMatch}}}},
		// 同一文档的 bank_list 即使有重复值，也只对对应题库计数一次。
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "question", Value: "$_id"}, {Key: "bank_id", Value: "$bank_list"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$_id.bank_id"}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	}
	cursor, err = db.Collection("question").Aggregate(ctx, pipeline)
	if err != nil {
		return result, fmt.Errorf("aggregate question counts: %w", err)
	}
	var counts []struct {
		BankID string `bson:"_id"`
		Count  int64  `bson:"count"`
	}
	if err = cursor.All(ctx, &counts); err != nil {
		return result, fmt.Errorf("decode question counts: %w", err)
	}
	for _, count := range counts {
		result.Counts[count.BankID] = count.Count
	}
	return result, nil
}

// Question 对应数据字典中的 question 集合，不包含用户动态字段。
type Question struct {
	ID            primitive.ObjectID `bson:"_id"`
	QuestionID    string             `bson:"question_id"`
	Title         string             `bson:"title"`
	Content       string             `bson:"content"`
	BankList      []string           `bson:"bank_list"`
	JobName       string             `bson:"job_name"`
	Difficulty    int                `bson:"difficulty"`
	Tags          []string           `bson:"tags"`
	Status        int                `bson:"status"`
	VIP           bool               `bson:"vip"`
	HotDegree     int                `bson:"hot_degree"`
	ViewCount     int                `bson:"view_count"`
	ThumbsUpCount int                `bson:"thumbs_up_count"`
	DislikeCount  int                `bson:"dislike_count"`
	Order         int64              `bson:"order"`
	CreateTime    time.Time          `bson:"create_time"`
	UpdateTime    time.Time          `bson:"update_time"`
}

// QuestionFilter 只描述持久化查询条件，不包含 HTTP 或游客展示逻辑。
// user_tag 是用户交互信息，本次游客查询不使用它。
type QuestionFilter struct {
	BankID     string
	Keyword    string
	Difficulty int
	Tags       []string
	JobName    string
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

func (r QuestionRepository) database() (*mongo.Database, error) {
	if r.db != nil {
		return r.db, nil
	}
	d := GetData()
	if d == nil || d.MongoDB == nil {
		return nil, errors.New("MongoDB is not initialized")
	}
	return d.MongoDB.Database(config.Conf.MongoDB.Database), nil
}

// FilterQuestions 统计符合条件的总数并查询当前页；不处理内容截断。
func (r QuestionRepository) FilterQuestions(ctx context.Context, req QuestionFilter) ([]Question, int64, error) {
	questions := make([]Question, 0)
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}
	if req.SortBy == "" {
		req.SortBy = "create_time"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}
	if req.Page < 1 || req.PageSize < 1 || int64(req.Page-1) > (1<<63-1)/int64(req.PageSize) {
		return nil, 0, errors.New("invalid question pagination")
	}
	switch req.SortBy {
	case "create_time", "view_count", "thumbs_up_count", "dislike_count":
	default:
		return nil, 0, errors.New("invalid question sort field")
	}
	direction := 1
	switch req.SortOrder {
	case "asc":
	case "desc":
		direction = -1
	default:
		return nil, 0, errors.New("invalid question sort order")
	}
	if req.Difficulty < 0 || req.Difficulty > 3 {
		return nil, 0, errors.New("invalid question difficulty")
	}
	filter := bson.D{{Key: "status", Value: QuestionStatusNormal}}
	if req.BankID != "" {
		filter = append(filter, bson.E{Key: "bank_list", Value: req.BankID})
	}
	if req.Keyword != "" {
		keyword := primitive.Regex{Pattern: regexp.QuoteMeta(req.Keyword), Options: "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{bson.D{{Key: "title", Value: keyword}}, bson.D{{Key: "content", Value: keyword}}}})
	}
	if req.Difficulty != 0 {
		filter = append(filter, bson.E{Key: "difficulty", Value: req.Difficulty})
	}
	if len(req.Tags) > 0 {
		filter = append(filter, bson.E{Key: "tags", Value: bson.D{{Key: "$all", Value: req.Tags}}})
	}
	if req.JobName != "" {
		filter = append(filter, bson.E{Key: "job_name", Value: req.JobName})
	}
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	collection := db.Collection("question")
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count questions: %w", err)
	}
	cursor, err := collection.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: req.SortBy, Value: direction}, {Key: "question_id", Value: 1}}).
		SetSkip(int64(req.Page-1)*int64(req.PageSize)).SetLimit(int64(req.PageSize)))
	if err != nil {
		return nil, 0, fmt.Errorf("find questions: %w", err)
	}
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, 0, fmt.Errorf("decode questions: %w", err)
	}
	return questions, total, nil
}

// ErrQuestionNotFound 表示题目不存在，与连接、查询和解码错误区分。
var ErrQuestionNotFound = errors.New("question not found")

// GetQuestionByID 只按业务 ID 查询，不自增浏览量，也不增加状态条件。
func (r QuestionRepository) GetQuestionByID(ctx context.Context, questionID string) (Question, error) {
	db, err := r.database()
	if err != nil {
		return Question{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var question Question
	err = db.Collection("question").FindOne(ctx, bson.D{{Key: "question_id", Value: questionID}}).Decode(&question)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Question{}, fmt.Errorf("get question detail: %w", ErrQuestionNotFound)
	}
	if err != nil {
		return Question{}, fmt.Errorf("get question detail: %w", err)
	}
	return question, nil
}

// GetHotQuestions 独立按 hot_degree 查询，与 FilterQuestions 的浏览量排序无关。
func (r QuestionRepository) GetHotQuestions(ctx context.Context, jobName string, limit int) ([]Question, error) {
	if limit == 0 {
		limit = 10
	}
	if limit < 1 {
		return nil, errors.New("hot question limit must be positive")
	}
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	filter := bson.D{{Key: "status", Value: QuestionStatusNormal}}
	if jobName != "" {
		filter = append(filter, bson.E{Key: "job_name", Value: jobName})
	}
	cursor, err := db.Collection("question").Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "hot_degree", Value: -1}, {Key: "question_id", Value: 1}}).
		SetLimit(int64(limit)).
		SetProjection(bson.D{{Key: "_id", Value: 0}, {Key: "question_id", Value: 1}, {Key: "bank_list", Value: 1}, {Key: "title", Value: 1}, {Key: "view_count", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find hot questions: %w", err)
	}
	questions := make([]Question, 0)
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, fmt.Errorf("decode hot questions: %w", err)
	}
	return questions, nil
}
