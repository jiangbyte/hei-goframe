package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"

	"hei-gin/config"
	ent "hei-gin/ent/gen"
)

var Client *ent.Client

func InitEnt() error {
	cfg := config.C.DB
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	sqldb.SetMaxOpenConns(cfg.PoolSize + cfg.MaxOverflow)
	sqldb.SetMaxIdleConns(cfg.PoolSize)
	sqldb.SetConnMaxLifetime(time.Duration(cfg.PoolRecycle) * time.Second)
	sqldb.SetConnMaxIdleTime(time.Duration(cfg.PoolRecycle) * time.Second)

	drv := entsql.OpenDB(dialect.MySQL, sqldb)
	Client = ent.NewClient(ent.Driver(drv))

	if err := sqldb.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	log.Println("[Database] MySQL connection verified")
	return nil
}

func Close() {
	if Client != nil {
		Client.Close()
		Client = nil
	}
}
