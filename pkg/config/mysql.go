package config

import (
	"fmt"
	"mygo_bangforai/api/model"
	"mygo_bangforai/api/errors"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitMySQL() error {
	dbConfig := GetDatabaseConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbConfig.Username,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.DBName,
	)
	Db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		err=errors.WrapError(err, errors.ConfigError, "数据库连接失败", "pkg/config.InitMySQL()")
		return err
	}
	sqlDB, err := Db.DB()
	if err != nil {
		err=errors.WrapError(err, errors.ConfigError, "数据库连接失败", "pkg/config.InitMySQL()")
		return err
	}
	// 使用配置文件中的连接池配置
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetime) * time.Second) // 连接最大生命周期

	DB = Db

	if err := autoMigrate(); err != nil {
		return errors.WrapError(err, errors.ConfigError, "数据库迁移失败", "pkg/config.InitMySQL()")
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}

func autoMigrate() error {
	// 自动迁移所有模型
	err := DB.AutoMigrate(
		&model.User{},
		&model.Session{},
		&model.ChatMessage{},
		&model.AIImageRecognition{},
		&model.Project{},
		&model.ProjectMember{},
	)

	if err != nil {
		return errors.WrapError(err, errors.ConfigError, "数据库迁移失败", "pkg/config.InitMySQL()")
	}

	return nil
}