package data

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"
)

type profileRows struct{ authUserRows }

func (*profileRows) Columns() []string {
	return []string{"user_id", "username", "nick_name", "avatar", "vip", "sex", "phone", "email", "introduction", "user_status", "user_type", "create_time", "update_time"}
}
func TestGetUserByIDLoadsProfileWithoutPassword(t *testing.T) {
	created := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	conn := &authSQLConn{rows: &profileRows{authUserRows{values: []driver.Value{"u1", "cat", "猫", "avatar", int64(1), int64(2), "phone", "email", "intro", int64(1), int64(2), created, updated}}}}
	user, err := authTestRepository(t, conn).GetUserByID(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if user.UserID != "u1" || user.NickName != "猫" || !user.VIP || user.Sex != 2 || user.Introduction != "intro" || !user.CreateTime.Equal(created) || !user.UpdateTime.Equal(updated) {
		t.Fatalf("incorrect profile %+v", user)
	}
	if !strings.Contains(conn.query, "FROM `t_user_info` WHERE user_id = ? LIMIT ?") || strings.Contains(conn.query, "password") || len(conn.args) != 2 || conn.args[0].Value != "u1" {
		t.Fatalf("unsafe profile query %s %+v", conn.query, conn.args)
	}
}
func TestGetUserByIDErrors(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, tc := range []struct {
		conn *authSQLConn
		want error
	}{
		{&authSQLConn{rows: &profileRows{}}, ErrUserNotFound}, {&authSQLConn{err: failure}, failure},
	} {
		if _, err := authTestRepository(t, tc.conn).GetUserByID(context.Background(), "missing"); !errors.Is(err, tc.want) {
			t.Fatalf("got %v want %v", err, tc.want)
		}
	}
}
