package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"mygo_bangforai/api/model"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type ApprovalRedisRepository interface {
	StagePreStoragePayload(ctx context.Context, msg *model.PreStorageMessage) error
	TakePreStoragePayload(ctx context.Context, userID, projectID uint) (*model.PreStorageMessage, error)
	SetUserProjectStatus(ctx context.Context, userID, projectID uint, status model.ChangeRequestStatus) error
	RemovePendingChangeForProject(ctx context.Context, userID, projectID uint) error
	ApplyApprovedVirtualFolderCache(ctx context.Context, projectID uint, msg *model.PreStorageMessage, tempToReal map[uint]uint, root *model.VirtualRoot) error
}

type approvalRedisRepository struct {
	rdb *redis.Client
}

func NewApprovalRedisRepository(rdb *redis.Client) ApprovalRedisRepository {
	return &approvalRedisRepository{rdb: rdb}
}

func prestoragePayloadKey(userID, projectID uint) string {
	return fmt.Sprintf("prestorage:payload:%d:%d", userID, projectID)
}

func (r *approvalRedisRepository) StagePreStoragePayload(ctx context.Context, msg *model.PreStorageMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal prestorage: %w", err)
	}
	return r.rdb.Set(ctx, prestoragePayloadKey(msg.UserID, msg.ProjectID), jsonData, 7*24*time.Hour).Err()
}

func (r *approvalRedisRepository) TakePreStoragePayload(ctx context.Context, userID, projectID uint) (*model.PreStorageMessage, error) {
	raw, err := r.rdb.GetDel(ctx, prestoragePayloadKey(userID, projectID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m model.PreStorageMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *approvalRedisRepository) SetUserProjectStatus(ctx context.Context, userID, projectID uint, status model.ChangeRequestStatus) error {
	key := fmt.Sprintf("userId:%d", userID)
	_, err := r.rdb.HSet(ctx, key, "project", projectID, "status", string(status)).Result()
	return err
}

func (r *approvalRedisRepository) RemovePendingChangeForProject(ctx context.Context, userID, projectID uint) error {
	listKey := fmt.Sprintf("user:%d:pending_updates", userID)
	vals, err := r.rdb.LRange(ctx, listKey, 0, -1).Result()
	if err != nil {
		return err
	}
	for _, item := range vals {
		var req model.ChangeRequest
		if json.Unmarshal([]byte(item), &req) != nil || req.ProjectID != projectID {
			continue
		}
		if _, err := r.rdb.LRem(ctx, listKey, 1, item).Result(); err != nil {
			return err
		}
		break
	}
	idxKey := fmt.Sprintf("project:%d:pending_users", projectID)
	_, err = r.rdb.SRem(ctx, idxKey, userID).Result()
	return err
}

func (r *approvalRedisRepository) ApplyApprovedVirtualFolderCache(ctx context.Context, projectID uint, msg *model.PreStorageMessage, tempToReal map[uint]uint, root *model.VirtualRoot) error {
	projectFoldersKey := fmt.Sprintf("project:%d:folders", projectID)
	projectKey := fmt.Sprintf("project:%d", projectID)
	dirty := false

	for _, op := range msg.Ops.Create {
		realID := tempToReal[op.TempID]
		if realID == 0 {
			continue
		}
		if err := r.bumpFolderVersion(ctx, projectFoldersKey, realID); err != nil {
			return err
		}
		pID, pType := ResolveParentRefForApproval(op.FatherID, op.FatherIDType, tempToReal, root)
		ft, fid := fatherCacheFields(pType, pID, root.ID)
		folderKey := fmt.Sprintf("virfolder:%d", realID)
		if err := r.rdb.HSet(ctx, folderKey, map[string]interface{}{
			"name":              op.Name,
			"fathertype":        ft,
			"fathervirfolderid": fid,
		}).Err(); err != nil {
			return err
		}
		dirty = true
	}

	for _, op := range msg.Ops.Move {
		if err := r.bumpFolderVersion(ctx, projectFoldersKey, op.ID); err != nil {
			return err
		}
		newPID, newPType := ResolveParentRefForApproval(op.NewFatherID, op.NewFatherIDType, tempToReal, root)
		ft, fid := fatherCacheFields(newPType, newPID, root.ID)
		folderKey := fmt.Sprintf("virfolder:%d", op.ID)
		if err := r.rdb.HSet(ctx, folderKey, map[string]interface{}{
			"fathertype":        ft,
			"fathervirfolderid": fid,
		}).Err(); err != nil {
			return err
		}
		dirty = true
	}

	for _, op := range msg.Ops.Rename {
		if err := r.bumpFolderVersion(ctx, projectFoldersKey, op.ID); err != nil {
			return err
		}
		folderKey := fmt.Sprintf("virfolder:%d", op.ID)
		if err := r.rdb.HSet(ctx, folderKey, "name", op.Name).Err(); err != nil {
			return err
		}
		dirty = true
	}

	for _, op := range msg.Ops.Delete {
		if err := r.rdb.HDel(ctx, projectFoldersKey, fmt.Sprintf("%d", op.ID)).Err(); err != nil {
			return err
		}
		folderKey := fmt.Sprintf("virfolder:%d", op.ID)
		filesKey := fmt.Sprintf("virfolder:%d:files", op.ID)
		if err := r.rdb.Del(ctx, folderKey, filesKey).Err(); err != nil {
			return err
		}
		dirty = true
	}

	if dirty {
		return r.rdb.HIncrBy(ctx, projectKey, "projectversion", 1).Err()
	}
	return nil
}

func fatherCacheFields(parentType string, parentID uint, rootID uint) (string, uint) {
	if parentType == "root" {
		return string(model.FatherTypeRoot), rootID
	}
	return string(model.FatherTypeVirtualFolder), parentID
}

func ResolveParentRefForApproval(id uint, idType string, tempToReal map[uint]uint, root *model.VirtualRoot) (uint, string) {
	if idType == "temp" {
		return tempToReal[id], "folder"
	}
	if id == root.ID {
		return root.ID, "root"
	}
	return id, "folder"
}

func (r *approvalRedisRepository) bumpFolderVersion(ctx context.Context, projectFoldersKey string, folderID uint) error {
	field := fmt.Sprintf("%d", folderID)
	cur, err := r.rdb.HGet(ctx, projectFoldersKey, field).Result()
	if err == redis.Nil {
		return r.rdb.HSet(ctx, projectFoldersKey, field, "1.0").Err()
	}
	if err != nil {
		return err
	}
	next, err := bumpSemverMinor(cur)
	if err != nil {
		return err
	}
	return r.rdb.HSet(ctx, projectFoldersKey, field, next).Err()
}

func bumpSemverMinor(version string) (string, error) {
	var major, minor int
	if n, _ := fmt.Sscanf(version, "%d.%d", &major, &minor); n == 2 {
		return fmt.Sprintf("%d.%d", major, minor+1), nil
	}
	i, err := strconv.Atoi(version)
	if err != nil {
		return "", fmt.Errorf("invalid version %q", version)
	}
	return fmt.Sprintf("%d.0", i+1), nil
}
