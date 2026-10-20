package model

// UserInfoData 按用户信息接口约定使用 snake_case，与登录响应区分。
type UserInfoData struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	NickName     string `json:"nick_name"`
	Avatar       string `json:"avatar"`
	VIP          bool   `json:"vip"`
	Sex          int    `json:"sex"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Introduction string `json:"introduction"`
	AvatarURL    string `json:"avatar_url"`
	UserStatus   int    `json:"user_status"`
	UserType     int    `json:"user_type"`
	CreateTime   string `json:"create_time"`
	UpdateTime   string `json:"update_time"`
}

type UserInfoResp struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data UserInfoData `json:"data"`
}
