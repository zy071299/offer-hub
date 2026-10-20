package data

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

// Hook 在 Redis 命令发送边界捕获实际 key 和过期参数。
type blacklistHook struct {
	args     []interface{}
	exists   int64
	inserted bool
	err      error
}

func (h *blacklistHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *blacklistHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *blacklistHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		h.args = cmd.Args()
		if h.err != nil {
			return h.err
		}
		switch c := cmd.(type) {
		case *redis.IntCmd:
			c.SetVal(h.exists)
		case *redis.BoolCmd:
			c.SetVal(h.inserted)
		}
		return nil
	}
}
func TestBlacklistUsesExactKeyAndExpiringAtomicWrite(t *testing.T) {
	h := &blacklistHook{inserted: true}
	client := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	defer client.Close()
	client.AddHook(h)
	repo := TokenBlacklistRepository{Client: client}
	exists, err := repo.IsTokenBlacklisted(context.Background(), "token")
	if err != nil || exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
	if len(h.args) != 2 || h.args[0] != "exists" || h.args[1] != "blacklist:token" {
		t.Fatalf("wrong lookup %+v", h.args)
	}
	added, err := repo.BlacklistToken(context.Background(), "token", 1500*time.Millisecond)
	if err != nil || !added {
		t.Fatalf("added=%v err=%v", added, err)
	}
	if len(h.args) != 6 || h.args[0] != "set" || h.args[1] != "blacklist:token" || h.args[2] != "1" || h.args[3] != "px" || h.args[4] != int64(1500) || h.args[5] != "nx" {
		t.Fatalf("wrong write %+v", h.args)
	}
	h.exists = 1
	exists, err = repo.IsTokenBlacklisted(context.Background(), "token")
	if err != nil || !exists {
		t.Fatal("blacklist entry ignored")
	}
	h.inserted = false
	added, err = repo.BlacklistToken(context.Background(), "token", time.Second)
	if err != nil || added {
		t.Fatal("duplicate write accepted")
	}
	h.err = errors.New("offline")
	if _, err := repo.BlacklistToken(context.Background(), "token", time.Second); err == nil {
		t.Fatal("write failure hidden")
	}
	if _, err := repo.IsTokenBlacklisted(context.Background(), "token"); err == nil {
		t.Fatal("read failure hidden")
	}
	for _, ttl := range []time.Duration{0, -time.Second} {
		if _, err := repo.BlacklistToken(context.Background(), "token", ttl); err == nil {
			t.Fatal("nonpositive TTL accepted")
		}
	}
}
