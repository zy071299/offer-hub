package model

type UpdateCommentReq struct {
	CommentID string `json:"comment_id" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

type UpdateCommentData struct {
	CommentID string `json:"comment_id"`
}

type UpdateCommentResp struct {
	Code int               `json:"code"`
	Msg  string            `json:"msg"`
	Data UpdateCommentData `json:"data"`
}

type DeleteCommentReq struct {
	CommentID string `json:"comment_id" binding:"required"`
}

// DeleteCommentResp 按删除接口文档不包含 data 字段。
type DeleteCommentResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type AddCommentReq struct {
	TargetType int    `json:"target_type"`
	TargetID   string `json:"target_id"`
	ParentID   string `json:"parent_id"`
	ReplyTo    string `json:"reply_to"`
	Content    string `json:"content"`
}

type AddCommentData struct {
	CommentID string      `json:"comment_id"`
	Comment   CommentInfo `json:"comment"`
}

type AddCommentResp struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data AddCommentData `json:"data"`
}

type ListCommentsReq struct {
	TargetType     int    `form:"target_type" json:"target_type"`
	TargetID       string `form:"target_id" json:"target_id"`
	ParentID       string `form:"parent_id" json:"parent_id"`
	SortBy         string `form:"sort_by" json:"sort_by"`
	SortOrder      string `form:"sort_order" json:"sort_order"`
	Page           int    `form:"page" json:"page"`
	PageSize       int    `form:"page_size" json:"page_size"`
	SubCommentPage int    `form:"sub_comment_page" json:"sub_comment_page"`
	SubCommentSize int    `form:"sub_comment_size" json:"sub_comment_size"`
	UserID         string `form:"-" json:"-"`
}

type CommentInfo struct {
	CommentID       string        `json:"comment_id"`
	UserID          string        `json:"user_id"`
	UserName        string        `json:"user_name"`
	UserAvatar      string        `json:"user_avatar"`
	Content         string        `json:"content"`
	ParentID        string        `json:"parent_id"`
	ReplyTo         string        `json:"reply_to"`
	ReplyToName     string        `json:"reply_to_name"`
	Status          int           `json:"status"`
	ThumbsUp        int           `json:"thumbs_up"`
	SubCommentTotal int64         `json:"sub_comment_total"`
	UserLiked       bool          `json:"user_liked"`
	SubComments     []CommentInfo `json:"sub_comments"`
	CreateTime      string        `json:"create_time"`
	UpdateTime      string        `json:"update_time"`
}

type ListCommentsData struct {
	Total int64         `json:"total"`
	List  []CommentInfo `json:"list"`
}

type ListCommentsResp struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data ListCommentsData `json:"data"`
}
