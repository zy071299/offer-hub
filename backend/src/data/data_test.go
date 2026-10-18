package data

import (
	"net"
	"offer-hub/backend/src/config"
	"testing"
)

func TestNewDataRejectsInvalidConfig(t *testing.T) {
	for _, conf := range []*config.TomlConfig{nil, {}, {MySQL: config.MySQLConfig{URL: "localhost:3306", DBName: "test"}}} {
		before := GetData()
		if _, err := NewData(conf); err == nil {
			t.Fatal("expected configuration error")
		}
		if GetData() != before {
			t.Fatal("failed initialization replaced global data")
		}
	}
}

func TestRedisRejectsInvalidConfig(t *testing.T) {
	before := GetRedis()
	if err := InitRedis(nil); err == nil {
		t.Fatal("expected nil configuration error")
	}
	if err := InitRedis(&config.TomlConfig{}); err == nil {
		t.Fatal("expected empty Redis address error")
	}
	if GetRedis() != before {
		t.Fatal("failed initialization replaced global Redis")
	}
}

func TestCloseUninitialized(t *testing.T) {
	if err := (&Data{}).Close(); err != nil {
		t.Fatal(err)
	}
	if err := CloseRedis(); err != nil {
		t.Fatal(err)
	}
}

func TestConnectionFailureDoesNotPublishGlobals(t *testing.T) {
	// 使用本地监听器立即拒绝握手，无需运行数据库。
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	conf := &config.TomlConfig{
		MySQL:   config.MySQLConfig{URL: listener.Addr().String(), DBName: "test"},
		MongoDB: config.MongoDBConfig{URL: "mongodb://127.0.0.1:27017", Database: "test"},
		Redis:   config.RedisConfig{URL: listener.Addr().String()},
	}
	before := GetData()
	if _, err := NewData(conf); err == nil {
		t.Fatal("expected MySQL handshake failure")
	}
	if GetData() != before {
		t.Fatal("failed connection published data")
	}
	redisBefore := GetRedis()
	if err := InitRedis(conf); err == nil {
		t.Fatal("expected Redis handshake failure")
	}
	if GetRedis() != redisBefore {
		t.Fatal("failed connection published Redis")
	}
}
