package repository

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Repositories 统一管理所有仓库
type Repositories struct {
	User            UserRepository
	Code            CodeRepository
	Project         ProjectRepository
	PostgresUser    *PostgresUserRepository
	PostgresProject *PostgresProjectRepository
	VirtualRoot     VirtualRootRepository
	VirtualFolder   VirtualFolderRepository
	MountRelation   MountRelationRepository
	RealFile        RealFileRepository
	CloudFile       CloudFileRepository
	CloudFileLocal  CloudFileLocalRepository
	Tag             TagRepository
	TagRelation     TagRelationRepository
	Test            *TestRepository
}

// NewRepositories 创建所有仓库实例
func NewRepositories(mysqlDB *gorm.DB, postgresDB *gorm.DB, rdb *redis.Client) *Repositories {
	return &Repositories{
		User:            NewUserRepository(mysqlDB),
		Code:            NewCodeRepository(rdb),
		Project:         NewProjectRepository(mysqlDB),
		PostgresUser:    NewPostgresUserRepository(postgresDB),
		PostgresProject: NewPostgresProjectRepository(postgresDB),
		VirtualRoot:     NewVirtualRootRepository(postgresDB),
		VirtualFolder:   NewVirtualFolderRepository(postgresDB),
		MountRelation:   NewMountRelationRepository(postgresDB),
		RealFile:        NewRealFileRepository(postgresDB),
		CloudFile:       NewCloudFileRepository(postgresDB),
		CloudFileLocal:  NewCloudFileLocalRepository(postgresDB),
		Tag:             NewTagRepository(postgresDB),
		TagRelation:     NewTagRelationRepository(postgresDB),
		Test:            NewTestRepository(postgresDB),
	}
}
