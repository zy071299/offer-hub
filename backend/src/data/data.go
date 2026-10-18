// Package data 管理数据库连接；初始化应在服务启动前执行。
package data

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"offer-hub/backend/src/config"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const connectionTimeout = 5 * time.Second

type Data struct {
	MySQL   *gorm.DB
	MongoDB *mongo.Client
}

var (
	mu          sync.RWMutex
	globalData  *Data
	redisClient *redis.Client
)

// GetData 在成功初始化前返回 nil。连接由本包统一关闭，调用方不应自行关闭。
func GetData() *Data {
	mu.RLock()
	defer mu.RUnlock()
	return globalData
}

// NewData 建立连接并 Ping；全部成功后才发布全局实例。
// 已初始化时返回错误，避免覆盖连接造成资源泄漏。
func NewData(conf *config.TomlConfig) (*Data, error) {
	if conf == nil {
		return nil, errors.New("database config is nil")
	}
	if _, _, err := net.SplitHostPort(conf.MySQL.URL); err != nil {
		return nil, errors.New("MySQL url must be host:port")
	}
	if conf.MySQL.DBName == "" {
		return nil, errors.New("MySQL db_name is required")
	}
	if !strings.HasPrefix(conf.MongoDB.URL, "mongodb://") && !strings.HasPrefix(conf.MongoDB.URL, "mongodb+srv://") {
		return nil, errors.New("MongoDB url must be a MongoDB URI")
	}
	if conf.MongoDB.Database == "" {
		return nil, errors.New("MongoDB database is required")
	}
	if conf.MongoDB.MaxPool > 0 && conf.MongoDB.MinPool > conf.MongoDB.MaxPool {
		return nil, errors.New("MongoDB min_pool exceeds max_pool")
	}
	mu.Lock()
	defer mu.Unlock()
	if globalData != nil {
		return nil, errors.New("data already initialized")
	}

	sqlConfig := mysqldriver.NewConfig()
	sqlConfig.Net = "tcp"
	sqlConfig.Addr = conf.MySQL.URL
	sqlConfig.User = conf.MySQL.User
	sqlConfig.Passwd = conf.MySQL.Pwd
	sqlConfig.DBName = conf.MySQL.DBName
	sqlConfig.ParseTime = true
	sqlConfig.Timeout = connectionTimeout
	sqlConfig.ReadTimeout = connectionTimeout
	sqlConfig.WriteTimeout = connectionTimeout
	sqlConfig.Params = map[string]string{"charset": "utf8mb4"}
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: sqlConfig.FormatDSN(), SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, fmt.Errorf("initialize MySQL: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get MySQL connection: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	err = sqlDB.PingContext(ctx)
	cancel()
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}

	mongoOptions := options.Client().ApplyURI(conf.MongoDB.URL).
		SetConnectTimeout(connectionTimeout).SetServerSelectionTimeout(connectionTimeout).
		SetMaxPoolSize(conf.MongoDB.MaxPool).SetMinPoolSize(conf.MongoDB.MinPool)
	if conf.MongoDB.User != "" || conf.MongoDB.Pwd != "" {
		credential := options.Credential{}
		if mongoOptions.Auth != nil {
			credential = *mongoOptions.Auth
		}
		credential.Username = conf.MongoDB.User
		credential.Password = conf.MongoDB.Pwd
		mongoOptions.SetAuth(credential)
	}
	ctx, cancel = context.WithTimeout(context.Background(), connectionTimeout)
	client, err := mongo.Connect(ctx, mongoOptions)
	cancel()
	if err == nil {
		ctx, cancel = context.WithTimeout(context.Background(), connectionTimeout)
		err = client.Ping(ctx, readpref.Primary())
		cancel()
	}
	if err != nil {
		_ = sqlDB.Close()
		if client != nil {
			_ = disconnectMongo(client)
		}
		return nil, fmt.Errorf("initialize MongoDB: %w", err)
	}
	globalData = &Data{MySQL: db, MongoDB: client}
	return globalData, nil
}

func disconnectMongo(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	return client.Disconnect(ctx)
}

// Close 在请求停止后关闭连接；不应与使用连接的业务并发执行。
func (d *Data) Close() error {
	if d == nil {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	var errs []error
	if d.MySQL != nil {
		sqlDB, err := d.MySQL.DB()
		if err == nil {
			err = sqlDB.Close()
		}
		errs = append(errs, err)
		d.MySQL = nil
	}
	if d.MongoDB != nil {
		errs = append(errs, disconnectMongo(d.MongoDB))
		d.MongoDB = nil
	}
	if globalData == d {
		globalData = nil
	}
	return errors.Join(errs...)
}

// InitRedis 独立初始化 Redis，不依赖 MySQL 或 MongoDB。
func InitRedis(conf *config.TomlConfig) error {
	if conf == nil {
		return errors.New("Redis config is nil")
	}
	if _, _, err := net.SplitHostPort(conf.Redis.URL); err != nil {
		return errors.New("Redis url must be host:port")
	}
	if conf.Redis.DB < 0 {
		return errors.New("Redis db must be non-negative")
	}
	mu.Lock()
	defer mu.Unlock()
	if redisClient != nil {
		return errors.New("Redis already initialized")
	}
	client := redis.NewClient(&redis.Options{
		Addr: conf.Redis.URL, Password: conf.Redis.Pwd, DB: conf.Redis.DB,
		DialTimeout: connectionTimeout, ReadTimeout: connectionTimeout, WriteTimeout: connectionTimeout,
		ContextTimeoutEnabled: true, MaxRetries: -1,
	})
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("ping Redis: %w", err)
	}
	redisClient = client
	return nil
}

func GetRedis() *redis.Client {
	mu.RLock()
	defer mu.RUnlock()
	return redisClient
}

// CloseRedis 在请求停止后关闭 Redis 连接。
func CloseRedis() error {
	mu.Lock()
	defer mu.Unlock()
	if redisClient == nil {
		return nil
	}
	err := redisClient.Close()
	redisClient = nil
	return err
}
