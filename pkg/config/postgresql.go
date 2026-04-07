package config

import (
	"fmt"
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var PostgresDB *gorm.DB

func InitPostgreSQL() error {
	pgConfig := GetPostgreSQLConfig()
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		pgConfig.Host,
		pgConfig.Port,
		pgConfig.Username,
		pgConfig.Password,
		pgConfig.DBName,
	)
	Db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		err = errors.WrapError(err, errors.ConfigError, "PostgreSQL连接失败", "pkg/config.InitPostgreSQL()")
		return err
	}
	sqlDB, err := Db.DB()
	if err != nil {
		err = errors.WrapError(err, errors.ConfigError, "PostgreSQL连接失败", "pkg/config.InitPostgreSQL()")
		return err
	}
	// 使用配置文件中的连接池配置
	sqlDB.SetMaxIdleConns(pgConfig.MaxIdleConns)                                    // 最大空闲连接数
	sqlDB.SetMaxOpenConns(pgConfig.MaxOpenConns)                                    // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Duration(pgConfig.ConnMaxLifetime) * time.Second) // 连接最大生命周期

	PostgresDB = Db

	if err := autoMigratePostgreSQL(); err != nil {
		return errors.WrapError(err, errors.ConfigError, "PostgreSQL数据库迁移失败", "pkg/config.InitPostgreSQL()")
	}

	return nil
}

func GetPostgresDB() *gorm.DB {
	return PostgresDB
}

func autoMigratePostgreSQL() error {
	// 自动迁移所有模型
	err := PostgresDB.AutoMigrate(
		&model.PostgresUser{},
		&model.PostgresProject{},
		&model.VirtualRoot{},
		&model.VirtualFolder{},
		&model.MountRelation{},
		&model.WorkSet{},
		&model.RealFile{},
		&model.CloudFile{},
		&model.CloudFileLocal{},
		&model.Tag{},
		&model.TagRelation{},
		&model.Test{},
	)

	if err != nil {
		return errors.WrapError(err, errors.ConfigError, "PostgreSQL数据库迁移失败", "pkg/config.autoMigratePostgreSQL()")
	}

	return nil
}
