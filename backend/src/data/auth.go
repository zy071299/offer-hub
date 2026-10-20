package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var ErrUsernameExists = errors.New("username already exists")

var ErrUserNotFound = errors.New("user not found")

// UserInfo 是鉴权使用的数据库字段，禁止直接作为 HTTP 响应返回。
type UserInfo struct {
	UserID       string `gorm:"column:user_id"`
	Username     string
	Password     string
	NickName     string
	Avatar       string
	Sex          int
	VIP          bool `gorm:"column:vip"`
	Phone        string
	Email        string
	UserStatus   int
	UserType     int
	Introduction string
	CreateTime   time.Time
	UpdateTime   time.Time
}

func (r AuthRepository) GetUserByUsername(ctx context.Context, username string) (UserInfo, error) {
	db, err := r.database(ctx)
	if err != nil {
		return UserInfo{}, err
	}
	var user UserInfo
	err = db.Table("t_user_info").Select("user_id", "username", "password", "nick_name", "avatar", "sex", "vip", "phone", "email", "user_status", "user_type").Where("username = ?", username).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return UserInfo{}, ErrUserNotFound
	}
	if err != nil {
		return UserInfo{}, fmt.Errorf("get user by username: %w", err)
	}
	return user, nil
}

type AuthRepository struct{ DB *gorm.DB }

func (r AuthRepository) database(ctx context.Context) (*gorm.DB, error) {
	db := r.DB
	if db == nil {
		if d := GetData(); d != nil {
			db = d.MySQL
		}
	}
	if db == nil {
		return nil, errors.New("MySQL is not initialized")
	}
	return db.WithContext(ctx), nil
}

func (r AuthRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	db, err := r.database(ctx)
	if err != nil {
		return false, err
	}
	var count int64
	if err := db.Table("t_user_info").Where("username = ?", username).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check username: %w", err)
	}
	return count > 0, nil
}

func (r AuthRepository) CreateUser(ctx context.Context, user *UserInfo) error {
	db, err := r.database(ctx)
	if err != nil {
		return err
	}
	err = db.Table("t_user_info").Create(map[string]interface{}{
		"user_id": user.UserID, "username": user.Username,
		"password": user.Password, "nick_name": user.NickName,
	}).Error
	// 预查询后仍可能发生并发注册，最终由 uk_username 保证唯一性。
	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 && strings.Contains(mysqlErr.Message, "uk_username") {
		return ErrUsernameExists
	}
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
