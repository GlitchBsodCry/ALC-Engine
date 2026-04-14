package repository

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Repositories 统一管理所有仓库
type Repositories struct {
	User                   UserRepository
	Code                   CodeRepository
	Project                ProjectRepository
	PostgresUser           *PostgresUserRepository
	PostgresProject        *PostgresProjectRepository
	VirtualRoot            VirtualRootRepository
	VirtualFolder          VirtualFolderRepository
	MountRelation          MountRelationRepository
	RealFile               RealFileRepository
	CloudFile              CloudFileRepository
	CloudFileLocal         CloudFileLocalRepository
	Tag                    TagRepository
	TagRelation            TagRelationRepository
	CallRelation           CallRelationRepository
	ChangeRequest          ChangeRequestRepository
	ApprovalRedis          ApprovalRedisRepository
	ApprovalPG             ApprovalPGRepository
	CloudFileApprovalRedis CloudFileApprovalRedisRepository
	AISession              AISessionRepository
	Test                   *TestRepository
}

// NewRepositories 创建所有仓库实例
func NewRepositories(mysqlDB *gorm.DB, postgresDB *gorm.DB, rdb *redis.Client) *Repositories {
	vr := NewVirtualRootRepository(postgresDB)
	vf := NewVirtualFolderRepository(postgresDB)
	mr := NewMountRelationRepository(postgresDB)

	return &Repositories{
		User:                   NewUserRepository(mysqlDB),
		Code:                   NewCodeRepository(rdb),
		Project:                NewProjectRepository(mysqlDB),
		PostgresUser:           NewPostgresUserRepository(postgresDB),
		PostgresProject:        NewPostgresProjectRepository(postgresDB),
		VirtualRoot:            vr,
		VirtualFolder:          vf,
		MountRelation:          mr,
		RealFile:               NewRealFileRepository(postgresDB),
		CloudFile:              NewCloudFileRepository(postgresDB),
		CloudFileLocal:         NewCloudFileLocalRepository(postgresDB),
		Tag:                    NewTagRepository(postgresDB),
		TagRelation:            NewTagRelationRepository(postgresDB),
		CallRelation:           NewCallRelationRepository(postgresDB),
		ChangeRequest:          NewChangeRequestRepository(rdb),
		ApprovalRedis:          NewApprovalRedisRepository(rdb),
		ApprovalPG:             NewApprovalPGRepository(postgresDB, vf, vr, mr),
		CloudFileApprovalRedis: NewCloudFileApprovalRedisRepository(rdb),
		AISession:              NewAISessionRepository(mysqlDB),
		Test:                   NewTestRepository(postgresDB),
	}
}
