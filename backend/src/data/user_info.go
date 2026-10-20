package data

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// GetUserByID 只查询展示资料，不读取密码哈希。
func (r AuthRepository) GetUserByID(ctx context.Context, userID string) (UserInfo, error) {
	db, err := r.database(ctx)
	if err != nil {
		return UserInfo{}, err
	}
	var user UserInfo
	err = db.Table("t_user_info").Select("user_id", "username", "nick_name", "avatar", "vip", "sex", "phone", "email", "introduction", "user_status", "user_type", "create_time", "update_time").Where("user_id = ?", userID).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return UserInfo{}, ErrUserNotFound
	}
	if err != nil {
		return UserInfo{}, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}
