package model

// PasswordRegisterReq 是用户名密码注册请求。
type PasswordRegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// PasswordLoginReq 是用户名密码登录请求。
type PasswordLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// PasswordLoginUserInfo 按认证接口约定使用 camelCase JSON 字段名。
type PasswordLoginUserInfo struct {
	UserID     string `json:"userId"`
	Username   string `json:"username"`
	NickName   string `json:"nickName"`
	Avatar     string `json:"avatar"`
	Sex        int    `json:"sex"`
	VIP        bool   `json:"vip"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	UserStatus int    `json:"userStatus"`
	UserType   int    `json:"userType"`
}

// PasswordLoginData 是登录响应的 data 字段。
type PasswordLoginData struct {
	Token    string                `json:"token"`
	UserInfo PasswordLoginUserInfo `json:"userInfo"`
}
