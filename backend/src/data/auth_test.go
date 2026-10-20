package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 在 SQL 驱动边界捕获实际 GORM 查询，不连接外部数据库。
type authSQLConnector struct{ conn *authSQLConn }

func (c authSQLConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (c authSQLConnector) Driver() driver.Driver                        { return authSQLDriver{} }

type authSQLDriver struct{}

func (authSQLDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type authSQLConn struct {
	rows  driver.Rows
	query string
	args  []driver.NamedValue
	count int64
	err   error
}

func (*authSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (*authSQLConn) Close() error              { return nil }
func (*authSQLConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }
func (c *authSQLConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	c.query = q
	c.args = args
	if c.err != nil {
		return nil, c.err
	}
	if c.rows != nil {
		return c.rows, nil
	}
	return &authCountRows{count: c.count}, nil
}

type authUserRows struct {
	values []driver.Value
	done   bool
}

func (*authUserRows) Columns() []string {
	return []string{"user_id", "username", "password", "nick_name", "avatar", "sex", "vip", "phone", "email", "user_status", "user_type"}
}
func (*authUserRows) Close() error { return nil }
func (r *authUserRows) Next(values []driver.Value) error {
	if r.done || r.values == nil {
		return io.EOF
	}
	copy(values, r.values)
	r.done = true
	return nil
}

func TestGetUserByUsernameLoadsLoginFields(t *testing.T) {
	conn := &authSQLConn{rows: &authUserRows{values: []driver.Value{"u1", "cat", "hash", "猫", "avatar", int64(2), int64(1), "phone", "email", int64(1), int64(2)}}}
	user, err := authTestRepository(t, conn).GetUserByUsername(context.Background(), "cat")
	if err != nil {
		t.Fatal(err)
	}
	if user != (UserInfo{UserID: "u1", Username: "cat", Password: "hash", NickName: "猫", Avatar: "avatar", Sex: 2, VIP: true, Phone: "phone", Email: "email", UserStatus: 1, UserType: 2}) {
		t.Fatalf("incorrect mapped user %+v", user)
	}
	if !strings.Contains(conn.query, "FROM `t_user_info` WHERE username = ? LIMIT ?") || len(conn.args) != 2 || conn.args[0].Value != "cat" || conn.args[1].Value != int64(1) {
		t.Fatalf("unexpected query %s %+v", conn.query, conn.args)
	}
}

func TestGetUserByUsernameDistinguishesMissingAndStorageFailure(t *testing.T) {
	failure := errors.New("storage offline")
	for _, tc := range []struct {
		conn *authSQLConn
		want error
	}{
		{&authSQLConn{rows: &authUserRows{}}, ErrUserNotFound},
		{&authSQLConn{err: failure}, failure},
	} {
		_, err := authTestRepository(t, tc.conn).GetUserByUsername(context.Background(), "cat")
		if !errors.Is(err, tc.want) {
			t.Fatalf("got %v want %v", err, tc.want)
		}
	}
}
func (c *authSQLConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	c.query = q
	c.args = args
	return authSQLResult{}, c.err
}

type authSQLResult struct{}

func (authSQLResult) LastInsertId() (int64, error) { return 1, nil }
func (authSQLResult) RowsAffected() (int64, error) { return 1, nil }

type authCountRows struct {
	count int64
	done  bool
}

func (*authCountRows) Columns() []string { return []string{"count"} }
func (*authCountRows) Close() error      { return nil }
func (r *authCountRows) Next(values []driver.Value) error {
	if r.done {
		return io.EOF
	}
	values[0] = r.count
	r.done = true
	return nil
}
func authTestRepository(t *testing.T, c *authSQLConn) AuthRepository {
	t.Helper()
	db := sql.OpenDB(authSQLConnector{conn: c})
	t.Cleanup(func() { _ = db.Close() })
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return AuthRepository{DB: gdb}
}

func TestUsernameLookupUsesUserTableAndBoundParameter(t *testing.T) {
	for _, count := range []int64{0, 1} {
		conn := &authSQLConn{count: count}
		repo := authTestRepository(t, conn)
		exists, err := repo.UsernameExists(context.Background(), "cat' OR 1=1 --")
		if err != nil || exists != (count > 0) {
			t.Fatalf("exists=%v err=%v", exists, err)
		}
		if conn.query != "SELECT count(*) FROM `t_user_info` WHERE username = ?" || len(conn.args) != 1 || conn.args[0].Value != "cat' OR 1=1 --" {
			t.Fatalf("unexpected query %s %+v", conn.query, conn.args)
		}
	}
}

func TestCreateUserPreservesDatabaseDefaultsAndMapsOnlyUsernameConflict(t *testing.T) {
	other := errors.New("connection lost")
	for _, tc := range []struct {
		name        string
		dbErr, want error
	}{
		{"success", nil, nil},
		{"username duplicate", &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'cat' for key 't_user_info.uk_username'"}, ErrUsernameExists},
		{"UUID duplicate", &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'id' for key 'uk_user_id'"}, nil},
		{"storage error", other, other},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := &authSQLConn{err: tc.dbErr}
			repo := authTestRepository(t, conn)
			err := repo.CreateUser(context.Background(), &UserInfo{UserID: "uuid", Username: "cat", Password: "hash", NickName: "cat"})
			if tc.name == "UUID duplicate" {
				if err == nil || errors.Is(err, ErrUsernameExists) {
					t.Fatalf("wrong duplicate mapping: %v", err)
				}
			} else if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if !strings.HasPrefix(conn.query, "INSERT INTO `t_user_info` (`nick_name`,`password`,`user_id`,`username`) VALUES (?,?,?,?)") {
				t.Fatalf("unexpected insert %s", conn.query)
			}
			want := []string{"cat", "hash", "uuid", "cat"}
			if len(conn.args) != len(want) {
				t.Fatalf("unexpected arguments %+v", conn.args)
			}
			for i, v := range want {
				if conn.args[i].Value != v {
					t.Fatalf("argument %d: %+v", i, conn.args[i])
				}
			}
		})
	}
}
