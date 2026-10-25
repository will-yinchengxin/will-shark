package core

import (
	"context"
	"database/sql"
	"testing"
	"willshark/app/modules/mysql/dao"
	"willshark/envconfig/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"
)

func mockNode(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	return mockNodeWith(t, false)
}

func mockNodeWith(t *testing.T, monitorPings bool) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(monitorPings))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// 读写分离：SELECT → 从库；写、事务、Master() → 主库
func TestReadWriteSplitting(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	primary, pm := mockNode(t)
	replica, rm := mockNode(t)

	db, err := newGorm(config.MySQL{Policy: "random"}, primary, nil, []*sql.DB{replica})
	if err != nil {
		t.Fatal(err)
	}

	rm.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "from-replica"))
	var u user
	if err := db.WithContext(context.Background()).First(&u).Error; err != nil {
		t.Fatal(err)
	}
	if u.Name != "from-replica" {
		t.Fatalf("read should hit replica, got %q", u.Name)
	}

	pm.ExpectBegin()
	pm.ExpectExec("INSERT INTO `user`").WillReturnResult(sqlmock.NewResult(2, 1))
	pm.ExpectCommit()
	if err := db.Create(&user{Name: "w"}).Error; err != nil {
		t.Fatal(err)
	}

	// 事务内的读也走主库
	pm.ExpectBegin()
	pm.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "from-primary"))
	pm.ExpectCommit()
	err = db.Transaction(func(tx *gorm.DB) error {
		var u user
		if err := tx.First(&u).Error; err != nil {
			return err
		}
		if u.Name != "from-primary" {
			t.Errorf("tx read should hit primary, got %q", u.Name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// 显式强制主库
	pm.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "master"))
	var m user
	if err := (&dao.MysqlPool{DB: db}).Master().First(&m).Error; err != nil || m.Name != "master" {
		t.Fatalf("Master() read: %v %q", err, m.Name)
	}

	for name, mock := range map[string]sqlmock.Sqlmock{"primary": pm, "replica": rm} {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

// 主主：写请求在主库与 sources 间轮询；无从库时读也在主库间分配
func TestMultiPrimaryRoundRobin(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	p1, m1 := mockNode(t)
	p2, m2 := mockNode(t)
	db, err := newGorm(config.MySQL{Policy: "round_robin"}, p1, []*sql.DB{p2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []sqlmock.Sqlmock{m1, m2, m1, m2} {
		m.ExpectExec("UPDATE `user`").WillReturnResult(sqlmock.NewResult(0, 1))
	}
	for i := 0; i < 4; i++ {
		if err := db.Exec("UPDATE `user` SET name = ?", "x").Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := m1.ExpectationsWereMet(); err != nil {
		t.Fatal("primary1:", err)
	}
	if err := m2.ExpectationsWereMet(); err != nil {
		t.Fatal("primary2:", err)
	}
}

func TestMySQLConnPingAndCloseAllNodes(t *testing.T) {
	p, pm := mockNodeWith(t, true)
	r, rm := mockNodeWith(t, true)
	conn := &MySQLConn{nodes: []mysqlNode{{role: "primary", addr: "p", db: p}, {role: "replica", addr: "r", db: r}}}

	pm.ExpectPing()
	rm.ExpectPing().WillReturnError(sql.ErrConnDone)
	if err := conn.Ping(context.Background()); err == nil {
		t.Fatal("ping should fail when a replica is down")
	}
	pm.ExpectClose()
	rm.ExpectClose()
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := pm.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if err := rm.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	var nilConn *MySQLConn
	if nilConn.Close() != nil {
		t.Fatal("nil Close should be no-op")
	}
}

func TestNewRedisModes(t *testing.T) {
	mr := miniredis.RunT(t)
	ctx := context.Background()

	cluster, err := NewRedis(ctx, config.Redis{Mode: config.RedisModeCluster, Addrs: []string{mr.Addr()}})
	if err != nil {
		t.Fatalf("cluster: %v", err)
	}
	defer cluster.Close()
	if err := cluster.Set(ctx, "k", "v", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if v, _ := cluster.Get(ctx, "k").Result(); v != "v" {
		t.Fatalf("cluster get = %q", v)
	}
	if _, ok := cluster.(*redis.ClusterClient); !ok {
		t.Fatalf("cluster mode should create *redis.ClusterClient, got %T", cluster)
	}

	if _, err := NewRedis(ctx, config.Redis{Mode: "bogus"}); err == nil {
		t.Fatal("unknown mode should fail")
	}
	// 哨兵不可达时应快速失败
	if _, err := NewRedis(ctx, config.Redis{Mode: config.RedisModeSentinel, Addrs: []string{"127.0.0.1:1"}, MasterName: "m", DialTimeout: 200e6}); err == nil {
		t.Fatal("unreachable sentinel should fail")
	}
}
