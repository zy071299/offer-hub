package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitParsesAllSections(t *testing.T) {
	old := Conf
	t.Cleanup(func() { Conf = old })
	t.Setenv("APP_ENV", "missing")
	path := filepath.Join(t.TempDir(), "custom.toml")
	content := `[COMMON]
port = 8180
open_tls = true
[MySQL]
url = "mysql-host:3306"
user = "mysql-user"
pwd = "mysql-password"
db_name = "offers"
[MongoDB]
url = "mongodb://localhost:27017"
user = "mongo-user"
pwd = "mongo-password"
database = "questions"
max_pool = 100
min_pool = 5
[Redis]
url = "localhost:6379"
pwd = "redis-password"
db = 2
[JWT]
secret = "test-secret"
expire = 24
token_cache_expire = 2
enable = true
[RateLimit]
enable = true
window_seconds = 60
max_requests = 20
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Init(path); err != nil {
		t.Fatal(err)
	}
	want := TomlConfig{
		Common:    CommonConfig{Port: 8180, OpenTLS: true},
		MySQL:     MySQLConfig{URL: "mysql-host:3306", User: "mysql-user", Pwd: "mysql-password", DBName: "offers"},
		MongoDB:   MongoDBConfig{URL: "mongodb://localhost:27017", User: "mongo-user", Pwd: "mongo-password", Database: "questions", MaxPool: 100, MinPool: 5},
		Redis:     RedisConfig{URL: "localhost:6379", Pwd: "redis-password", DB: 2},
		JWT:       JWTConfig{Secret: "test-secret", Expire: 24, TokenCacheExpire: 2, Enable: true},
		RateLimit: RateLimitConfig{Enable: true, WindowSeconds: 60, MaxRequests: 20},
	}
	if Conf != want {
		t.Fatal("parsed configuration differs from expected values")
	}
}

func TestInitSelectsEnvironment(t *testing.T) {
	old := Conf
	t.Cleanup(func() { Conf = old })
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	for _, tc := range []struct {
		env, file string
		port      int
	}{
		{"", "config-test.toml", 8180},
		{"prod", "config-prod.toml", 9090},
	} {
		t.Run(tc.file, func(t *testing.T) {
			t.Setenv("APP_ENV", tc.env)
			content := "[COMMON]\nport = 8180\n"
			if tc.port == 9090 {
				content = "[COMMON]\nport = 9090\n"
			}
			if err := os.WriteFile(tc.file, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if err := Init(""); err != nil {
				t.Fatal(err)
			}
			if Conf.Common.Port != tc.port {
				t.Fatalf("port = %d, want %d", Conf.Common.Port, tc.port)
			}
		})
	}
}

func TestInitFailurePreservesConf(t *testing.T) {
	old := Conf
	t.Cleanup(func() { Conf = old })
	Conf = TomlConfig{Common: CommonConfig{Port: 8180}}
	for _, content := range []string{"invalid", "[COMMON]\nport = 'wrong type'"} {
		path := filepath.Join(t.TempDir(), "bad.toml")
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if err := Init(path); err == nil {
			t.Fatal("expected parsing error")
		}
		if Conf.Common.Port != 8180 {
			t.Fatal("failed load changed Conf")
		}
	}
	if err := Init(filepath.Join(t.TempDir(), "missing.toml")); err == nil {
		t.Fatal("expected file error")
	}
	if Conf.Common.Port != 8180 {
		t.Fatal("missing file changed Conf")
	}
}
