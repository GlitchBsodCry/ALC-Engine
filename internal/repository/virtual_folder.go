package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

type VirtualFolderRepository interface {
	CreateVirtualFolder(ctx context.Context, folder *model.VirtualFolder) (uint, error)
	GetVirtualFolderByID(ctx context.Context, folderID uint) (*model.VirtualFolder, error)
	GetVirtualFoldersByIDs(ctx context.Context, folderIDs []uint) ([]model.VirtualFolder, error)
	GetVirtualFoldersByRootID(ctx context.Context, rootID uint) ([]model.VirtualFolder, error)
	GetVirtualFoldersByParentID(ctx context.Context, parentID uint) ([]model.VirtualFolder, error)
	UpdateVirtualFolder(ctx context.Context, folder *model.VirtualFolder) error
	DeleteVirtualFolder(ctx context.Context, folderID uint) error
}

type virtualFolderRepository struct {
	db *gorm.DB
}

func NewVirtualFolderRepository(db *gorm.DB) VirtualFolderRepository {
	return &virtualFolderRepository{
		db: db,
	}
}

func (r *virtualFolderRepository) CreateVirtualFolder(ctx context.Context, folder *model.VirtualFolder) (uint, error) {
	err := r.db.WithContext(ctx).Create(folder).Error
	if err != nil {
		return 0, err
	}
	return folder.ID, nil
}

func (r *virtualFolderRepository) GetVirtualFolderByID(ctx context.Context, folderID uint) (*model.VirtualFolder, error) {
	var folder model.VirtualFolder
	err := r.db.WithContext(ctx).
		Where("id = ?", folderID).
		First(&folder).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func (r *virtualFolderRepository) GetVirtualFoldersByRootID(ctx context.Context, rootID uint) ([]model.VirtualFolder, error) {
	var folders []model.VirtualFolder
	err := r.db.WithContext(ctx).
		Where("root_id = ?", rootID).
		Find(&folders).Error
	return folders, err
}

func (r *virtualFolderRepository) GetVirtualFoldersByParentID(ctx context.Context, parentID uint) ([]model.VirtualFolder, error) {
	var folders []model.VirtualFolder
	// 这里需要通过MountRelation来查询子文件夹
	// 首先查询与父文件夹相关的MountRelation
	var mountRelations []model.MountRelation
	err := r.db.WithContext(ctx).
		Where("parent_id = ? AND parent_type = 'folder' AND child_type = 'folder'", parentID).
		Find(&mountRelations).Error
	if err != nil {
		return nil, err
	}

	// 提取子文件夹ID
	var childFolderIDs []uint
	for _, relation := range mountRelations {
		childFolderIDs = append(childFolderIDs, relation.ChildID)
	}

	// 查询子文件夹
	if len(childFolderIDs) > 0 {
		err = r.db.WithContext(ctx).
			Where("id IN ?", childFolderIDs).
			Find(&folders).Error
	}

	return folders, err
}

func (r *virtualFolderRepository) UpdateVirtualFolder(ctx context.Context, folder *model.VirtualFolder) error {
	return r.db.WithContext(ctx).Save(folder).Error
}

func (r *virtualFolderRepository) DeleteVirtualFolder(ctx context.Context, folderID uint) error {
	return r.db.WithContext(ctx).Delete(&model.VirtualFolder{}, folderID).Error
}

func (r *virtualFolderRepository) GetVirtualFoldersByIDs(ctx context.Context, folderIDs []uint) ([]model.VirtualFolder, error) {
	var folders []model.VirtualFolder
	err := r.db.WithContext(ctx).Where("id IN ?", folderIDs).Find(&folders).Error
	return folders, err
}
