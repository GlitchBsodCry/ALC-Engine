package model

import (
	"time"

	"gorm.io/gorm"
)

type PostgresUser struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type PostgresProject struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type VirtualRoot struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	UserID    *uint  `json:"user_id" gorm:"default:null"`
	ProjectId *uint  `json:"project_id" gorm:"default:null"`
	OwnerID   uint   `json:"owner_id" gorm:"not null"`
	Type      string `json:"type" gorm:"size:20;not null"` // project or user
}

type VirtualFolder struct {
	ID        uint `json:"id" gorm:"primaryKey"`
	UserID    uint `json:"user_id" gorm:"not null"`    // 虚文件所属用户
	ProjectId uint `json:"project_id" gorm:"not null"` // 虚文件所属用户的项目
	RootID    uint `json:"root_id" gorm:"not null"`    // 根文件夹ID

	Name      string         `json:"name" gorm:"size:50;not null"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type RealFile struct { //已淘汰
	ID        uint `json:"id" gorm:"primaryKey"`
	UserID    uint `json:"user_id" gorm:"not null"`
	ProjectId uint `json:"project_id" gorm:"not null"`
	RootID    uint `json:"root_id" gorm:"not null"` // 根文件夹ID

	Name string `json:"name" gorm:"size:50;not null"`
	Path string `json:"path" gorm:"size:255;not null"`

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type NewRealFile struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	UserID uint `json:"user_id" gorm:"not null"`

	Name string `json:"name" gorm:"size:50;not null"`
	Path string `json:"path" gorm:"size:255;not null"`
	Hash string `json:"hash" gorm:"size:255"` // 文件哈希，用于上传校验

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type NewCloudFile struct {
	ID            uint `json:"id" gorm:"primaryKey"`
	NewRealFileID uint `json:"new_real_file_id" gorm:"not null;index"` // 关联权威文件
	ProjectId     uint `json:"project_id" gorm:"not null"`
	RootID        uint `json:"root_id" gorm:"not null"` // 根文件夹ID

	CloudStroageKey string `json:"cloud_storage_key" gorm:"size:255;not null;unique"` // MinIO键
	Bucket          string `json:"bucket" gorm:"size:50;not null"`                    // MinIO桶
	MimeType        string `json:"mime_type" gorm:"size:127;not null"`                // MIME类型，如 “image/png” 从可信源minio异步获取

	Name string `json:"name" gorm:"size:50;not null"`
	Hash string `json:"hash" gorm:"size:255;not null"` // 文件哈希值，用于校验文件完整性

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type NewCloudFileLocal struct {
	ID             uint   `json:"id" gorm:"primaryKey"`
	UserID         uint   `json:"user_id" gorm:"not null;index"`           // 本地用户
	NewCloudFileID uint   `json:"new_cloud_file_id" gorm:"not null;index"` // 关联云文件
	LocalPath      string `json:"local_path" gorm:"size:255;not null"`     // 本地存储路径
	ETag           string `json:"etag" gorm:"size:255"`                    // 最后一次同步时云端文件的ETag

	LastSync  time.Time `json:"last_sync" gorm:"autoUpdateTime"` // 最后同步时间
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CloudFileLocal struct { //已淘汰
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"user_id" gorm:"not null;index"`       // 本地用户ID
	CloudFileID uint      `json:"cloud_file_id" gorm:"not null;index"` // 关联的云文件ID
	LocalPath   string    `json:"local_path" gorm:"size:255;not null"` // 本地存储路径
	LastSync    time.Time `json:"last_sync" gorm:"autoUpdateTime"`     // 最后同步时间
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CloudFile struct { //已淘汰
	ID        uint `json:"id" gorm:"primaryKey"`
	UserID    uint `json:"user_id" gorm:"not null"` //上传者ID
	ProjectId uint `json:"project_id" gorm:"not null"`
	RootID    uint `json:"root_id" gorm:"not null"` // 根文件夹ID

	Name string `json:"name" gorm:"size:50;not null"`
	Path string `json:"path" gorm:"size:255;not null"`
	Hash string `json:"hash" gorm:"size:255;not null"` // 文件哈希值，用于校验文件完整性

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type MountRelation struct {
	ID       uint `json:"id" gorm:"primaryKey"`
	ParentID uint `json:"parent_id" gorm:"not null;index"` // 文件不可以是父节点
	ChildID  uint `json:"child_id" gorm:"not null;index"`  //根目录不可能是子节点

	ParentType string `json:"parent_type" gorm:"default:null"` // 父文件夹类型,root或folder
	ChildType  string `json:"child_type" gorm:"default:null"`  // 子文件夹类型,file或folder

	RelationType string `json:"relation_type" gorm:"size:20;default:null"` // 关系类型,"mount"(挂载) 或 "call"(调用)，调用只可用于虚资源之间

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type WorkSet struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"not null;index"`
	RootID          uint      `json:"root_id" gorm:"not null;index"`               // 所属根目录
	VirtualFolderID uint      `json:"virtual_folder_id" gorm:"default:null;index"` // 所属虚资源
	Name            string    `json:"name" gorm:"size:50;not null"`
	IsActive        bool      `json:"is_active" gorm:"default:false"` // 是否为当前活跃工作集
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Tag struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ProjectId uint           `json:"project_id" gorm:"not null;index"`
	Name      string         `json:"name" gorm:"size:50;not null;index"`
	Color     string         `json:"color" gorm:"size:20;default:'#6c757d'"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type TagRelation struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	ProjectId       uint           `json:"project_id" gorm:"not null;index"`
	TagID           uint           `json:"tag_id" gorm:"not null;index"`
	VirtualFolderID uint           `json:"virtual_folder_id" gorm:"not null;index"`
	CreatedAt       time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// CallRelation 虚拟文件夹调用关系
type CallRelation struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	ProjectId      uint           `json:"project_id" gorm:"not null;index"`
	CallerFolderID uint           `json:"caller_folder_id" gorm:"not null;index"` // 调用者文件夹ID
	CalledFolderID uint           `json:"called_folder_id" gorm:"not null;index"` // 被调用者文件夹ID
	CreatedAt      time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (PostgresUser) TableName() string {
	return "postgres_users"
}

func (CloudFileLocal) TableName() string {
	return "cloud_file_locals"
}

func (VirtualFolder) TableName() string {
	return "virtual_folders"
}

func (MountRelation) TableName() string {
	return "mount_relations"
}

func (WorkSet) TableName() string {
	return "work_sets"
}

func (RealFile) TableName() string {
	return "real_files"
}

func (CloudFile) TableName() string {
	return "cloud_files"
}

func (VirtualRoot) TableName() string {
	return "virtual_roots"
}

func (Tag) TableName() string {
	return "tags"
}

func (TagRelation) TableName() string {
	return "tag_relations"
}

func (CallRelation) TableName() string {
	return "call_relations"
}
