package repository

import (
	"context"
	"mygo_bangforai/api/model"

	"gorm.io/gorm"
)

// ApprovalPGRepository 封装审批批量操作相关的数据库操作
// 将多个Repository的操作聚合到一起，简化Service层调用
type ApprovalPGRepository interface {
	// ExecuteApprovedOps 执行批准的批量操作，返回临时ID到真实ID的映射和项目根目录
	ExecuteApprovedOps(ctx context.Context, userID uint, msg *model.PreStorageMessage) (map[uint]uint, *model.VirtualRoot, error)
}

type approvalPGRepository struct {
	db     *gorm.DB
	vfRepo VirtualFolderRepository
	vrRepo VirtualRootRepository
	mrRepo MountRelationRepository
}

func NewApprovalPGRepository(db *gorm.DB, vfRepo VirtualFolderRepository, vrRepo VirtualRootRepository, mrRepo MountRelationRepository) ApprovalPGRepository {
	return &approvalPGRepository{
		db:     db,
		vfRepo: vfRepo,
		vrRepo: vrRepo,
		mrRepo: mrRepo,
	}
}

func (r *approvalPGRepository) ExecuteApprovedOps(ctx context.Context, userID uint, msg *model.PreStorageMessage) (map[uint]uint, *model.VirtualRoot, error) {
	// 获取项目根目录
	root, err := r.vrRepo.GetVirtualRootByProjectID(ctx, msg.ProjectID)
	if err != nil {
		return nil, nil, err
	}

	tempToReal := make(map[uint]uint)

	// 开启事务
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, nil, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 使用事务上下文执行操作
	txCtx := context.WithValue(ctx, "tx", tx)

	// 批量执行创建操作（方案四：延迟关系建立 + 批量插入）
	if len(msg.Ops.Create) > 0 {
		if err := r.batchCreate(txCtx, userID, msg.ProjectID, msg.Ops.Create, root, tempToReal); err != nil {
			tx.Rollback()
			return nil, nil, err
		}
	}

	// 批量执行移动操作
	if len(msg.Ops.Move) > 0 {
		if err := r.batchMove(txCtx, msg.Ops.Move, tempToReal, root); err != nil {
			tx.Rollback()
			return nil, nil, err
		}
	}

	// 批量执行重命名操作
	if len(msg.Ops.Rename) > 0 {
		if err := r.batchRename(txCtx, msg.Ops.Rename); err != nil {
			tx.Rollback()
			return nil, nil, err
		}
	}

	// 批量执行删除操作
	if len(msg.Ops.Delete) > 0 {
		if err := r.batchDelete(txCtx, msg.Ops.Delete); err != nil {
			tx.Rollback()
			return nil, nil, err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, nil, err
	}

	return tempToReal, root, nil
}

// batchCreate 批量创建文件夹（方案四：延迟关系建立 + 批量插入）
// 步骤：
// 1. 批量插入所有新文件夹（不处理父子关系）
// 2. 建立 temp_id -> real_id 的映射
// 3. 批量创建 MountRelation 记录
func (r *approvalPGRepository) batchCreate(ctx context.Context, userID, projectID uint, creates []model.PreCreateOp, root *model.VirtualRoot, tempToReal map[uint]uint) error {
	tx := r.db.WithContext(ctx)

	// 步骤1：批量插入所有文件夹
	folders := make([]model.VirtualFolder, len(creates))
	for i, op := range creates {
		folders[i] = model.VirtualFolder{
			UserID:    userID,
			ProjectId: projectID,
			RootID:    root.ID,
			Name:      op.Name,
		}
	}

	if err := tx.CreateInBatches(folders, 100).Error; err != nil {
		return err
	}

	// 步骤2：建立 temp_id -> real_id 映射
	for i, op := range creates {
		tempToReal[op.TempID] = folders[i].ID
	}

	// 步骤3：批量创建 MountRelation
	relations := make([]model.MountRelation, len(creates))
	for i, op := range creates {
		parentID, parentKind := ResolveParentRefForApproval(op.FatherID, op.FatherIDType, tempToReal, root)

		relations[i] = model.MountRelation{
			ParentID:     parentID,
			ChildID:      folders[i].ID,
			ParentType:   parentKind,
			ChildType:    "folder",
			RelationType: "mount",
		}
	}

	if err := tx.CreateInBatches(relations, 100).Error; err != nil {
		return err
	}

	return nil
}

// batchMove 批量移动文件夹
func (r *approvalPGRepository) batchMove(ctx context.Context, moves []model.PreMoveOp, tempToReal map[uint]uint, root *model.VirtualRoot) error {
	tx := r.db.WithContext(ctx)

	// 收集需要删除的旧关系和需要创建的新关系
	var relationsToDelete []model.MountRelation
	var relationsToCreate []model.MountRelation

	for _, op := range moves {
		// 获取现有关系
		rels, err := r.mrRepo.GetMountRelationsByChildID(ctx, op.ID, "folder")
		if err != nil {
			return err
		}

		// 标记需要删除的挂载关系
		for _, rel := range rels {
			if rel.RelationType == "mount" {
				relationsToDelete = append(relationsToDelete, rel)
			}
		}

		// 构建新关系
		newPID, newPType := ResolveParentRefForApproval(op.NewFatherID, op.NewFatherIDType, tempToReal, root)
		relationsToCreate = append(relationsToCreate, model.MountRelation{
			ParentID:     newPID,
			ChildID:      op.ID,
			ParentType:   newPType,
			ChildType:    "folder",
			RelationType: "mount",
		})
	}

	// 批量删除旧关系
	if len(relationsToDelete) > 0 {
		for _, rel := range relationsToDelete {
			if err := tx.Where("parent_id = ? AND child_id = ?", rel.ParentID, rel.ChildID).Delete(&model.MountRelation{}).Error; err != nil {
				return err
			}
		}
	}

	// 批量创建新关系
	if len(relationsToCreate) > 0 {
		if err := tx.CreateInBatches(relationsToCreate, 100).Error; err != nil {
			return err
		}
	}

	return nil
}

// batchRename 批量重命名文件夹
func (r *approvalPGRepository) batchRename(ctx context.Context, renames []model.PreRenameOp) error {
	tx := r.db.WithContext(ctx)

	// 收集需要更新的文件夹ID和新名称
	folderIDs := make([]uint, len(renames))
	nameMap := make(map[uint]string)

	for i, op := range renames {
		folderIDs[i] = op.ID
		nameMap[op.ID] = op.Name
	}

	// 批量获取文件夹
	var folders []model.VirtualFolder
	if err := tx.Where("id IN ?", folderIDs).Find(&folders).Error; err != nil {
		return err
	}

	// 更新名称
	for i := range folders {
		folders[i].Name = nameMap[folders[i].ID]
	}

	// 批量更新
	if err := tx.Save(&folders).Error; err != nil {
		return err
	}

	return nil
}

// batchDelete 批量删除文件夹
func (r *approvalPGRepository) batchDelete(ctx context.Context, deletes []model.PreDeleteOp) error {
	tx := r.db.WithContext(ctx)

	// 收集需要删除的文件夹ID
	folderIDs := make([]uint, len(deletes))
	for i, op := range deletes {
		folderIDs[i] = op.ID
	}

	// 先删除相关的挂载关系
	if err := tx.Where("child_id IN ? AND child_type = 'folder'", folderIDs).Delete(&model.MountRelation{}).Error; err != nil {
		return err
	}

	// 批量删除文件夹
	if err := tx.Delete(&model.VirtualFolder{}, folderIDs).Error; err != nil {
		return err
	}

	return nil
}
