package service

import (
	"context"
	"fmt"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/repository"
	"sync"
	"time"
)

type TagService struct {
	TagRepo           repository.TagRepository
	TagRelationRepo   repository.TagRelationRepository
	VirtualFolderRepo repository.VirtualFolderRepository
	ProjectRepo       repository.ProjectRepository

	// 缓存
	tagCache           map[string][]model.Tag
	virtualFolderCache map[string][]model.VirtualFolder
	cacheMutex         sync.RWMutex
	cacheExpiry        time.Duration
}

func NewTagService(
	tagRepo repository.TagRepository,
	tagRelationRepo repository.TagRelationRepository,
	virtualFolderRepo repository.VirtualFolderRepository,
	projectRepo repository.ProjectRepository,
) *TagService {
	return &TagService{
		TagRepo:           tagRepo,
		TagRelationRepo:   tagRelationRepo,
		VirtualFolderRepo: virtualFolderRepo,
		ProjectRepo:       projectRepo,
		// 初始化缓存
		tagCache:           make(map[string][]model.Tag),
		virtualFolderCache: make(map[string][]model.VirtualFolder),
		cacheExpiry:        10 * time.Minute, // 缓存过期时间
	}
}

// CreateTag 创建标签
func (s *TagService) CreateTag(ctx context.Context, userID uint, projectID uint, name string, color string) error {
	// 检查用户是否为项目成员
	isMember, err := s.ProjectRepo.IsProjectMember(ctx, projectID, userID)
	if err != nil || !isMember {
		return nil // 权限不足
	}

	// 创建标签
	tag := &model.Tag{
		ProjectId: projectID,
		Name:      name,
		Color:     color,
	}

	// 清除相关缓存
	s.clearCache()

	return s.TagRepo.CreateTag(ctx, tag)
}

// GetTagsByProjectID 获取项目的所有标签
func (s *TagService) GetTagsByProjectID(ctx context.Context, userID uint, projectID uint) ([]model.Tag, error) {
	// 检查用户是否为项目成员
	isMember, err := s.ProjectRepo.IsProjectMember(ctx, projectID, userID)
	if err != nil || !isMember {
		return nil, nil // 权限不足
	}

	return s.TagRepo.GetTagsByProjectID(ctx, projectID)
}

// UpdateTag 更新标签
func (s *TagService) UpdateTag(ctx context.Context, userID uint, tagID uint, name string, color string) error {
	// 检查标签是否存在
	tag, err := s.TagRepo.GetTagByID(ctx, tagID)
	if err != nil {
		return err
	}

	// 检查用户是否为项目成员
	isMember, err := s.ProjectRepo.IsProjectMember(ctx, tag.ProjectId, userID)
	if err != nil || !isMember {
		return nil // 权限不足
	}

	// 更新标签
	tag.Name = name
	tag.Color = color

	// 清除相关缓存
	s.clearCache()

	return s.TagRepo.UpdateTag(ctx, tag)
}

// DeleteTag 删除标签
func (s *TagService) DeleteTag(ctx context.Context, userID uint, tagID uint) error {
	// 检查标签是否存在
	tag, err := s.TagRepo.GetTagByID(ctx, tagID)
	if err != nil {
		return err
	}

	// 检查用户是否为项目成员
	isMember, err := s.ProjectRepo.IsProjectMember(ctx, tag.ProjectId, userID)
	if err != nil || !isMember {
		return nil // 权限不足
	}

	// 清除相关缓存
	s.clearCache()

	// 直接删除标签，保留标签关系
	// 这样当标签被删除后，关系仍然存在，但标签会显示为nil
	return s.TagRepo.DeleteTag(ctx, tagID)
}

// AddTagToVirtualFolder 为虚拟文件夹添加标签
func (s *TagService) AddTagToVirtualFolder(ctx context.Context, userID uint, tagID uint, virtualFolderID uint) error {
	// 检查标签是否存在
	tag, err := s.TagRepo.GetTagByID(ctx, tagID)
	if err != nil {
		return err
	}

	// 检查虚拟文件夹是否存在
	virtualFolder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, virtualFolderID)
	if err != nil {
		return err
	}

	// 检查标签和虚拟文件夹是否属于同一个项目
	if tag.ProjectId != virtualFolder.ProjectId {
		return nil // 标签和虚拟文件夹不属于同一个项目
	}

	// 检查用户是否为项目成员
	isMember, err := s.ProjectRepo.IsProjectMember(ctx, tag.ProjectId, userID)
	if err != nil || !isMember {
		return nil // 权限不足
	}

	// 创建标签关系
	relation := &model.TagRelation{
		ProjectId:       tag.ProjectId,
		TagID:           tagID,
		VirtualFolderID: virtualFolderID,
	}

	// 清除相关缓存
	s.clearCache()

	return s.TagRelationRepo.CreateTagRelation(ctx, relation)
}

// RemoveTagFromVirtualFolder 从虚拟文件夹移除标签
func (s *TagService) RemoveTagFromVirtualFolder(ctx context.Context, userID uint, tagID uint, virtualFolderID uint) error {
	// 检查标签是否存在
	tag, err := s.TagRepo.GetTagByID(ctx, tagID)
	if err != nil {
		return err
	}

	// 检查虚拟文件夹是否存在
	virtualFolder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, virtualFolderID)
	if err != nil {
		return err
	}

	// 检查标签和虚拟文件夹是否属于同一个项目
	if tag.ProjectId != virtualFolder.ProjectId {
		return nil // 标签和虚拟文件夹不属于同一个项目
	}

	// 检查用户是否为项目成员
	isMember, err := s.ProjectRepo.IsProjectMember(ctx, tag.ProjectId, userID)
	if err != nil || !isMember {
		return nil // 权限不足
	}

	// 清除相关缓存
	s.clearCache()

	// 删除标签关系
	return s.TagRelationRepo.DeleteTagRelation(ctx, tagID, virtualFolderID)
}

// GetTagsByVirtualFolderID 获取虚拟文件夹的标签
func (s *TagService) GetTagsByVirtualFolderID(ctx context.Context, userID uint, virtualFolderID uint) ([]model.Tag, error) {
	// 检查虚拟文件夹是否存在
	virtualFolder, err := s.VirtualFolderRepo.GetVirtualFolderByID(ctx, virtualFolderID)
	if err != nil {
		return nil, err
	}

	// 检查用户是否为项目成员
	isMember, err := s.ProjectRepo.IsProjectMember(ctx, virtualFolder.ProjectId, userID)
	if err != nil || !isMember {
		return nil, nil // 权限不足
	}

	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("tags_by_folder:%d", virtualFolderID)
	if cached, found := s.getTagFromCache(cacheKey); found {
		return cached, nil
	}

	// 获取标签关系
	relations, err := s.TagRelationRepo.GetTagRelationsByVirtualFolderID(ctx, virtualFolderID)
	if err != nil {
		return nil, err
	}

	// 提取标签ID
	var tagIDs []uint
	for _, relation := range relations {
		tagIDs = append(tagIDs, relation.TagID)
	}

	// 批量获取标签
	var tags []model.Tag
	if len(tagIDs) > 0 {
		tags, err = s.TagRepo.GetTagsByIDs(ctx, tagIDs)
		if err != nil {
			return nil, err
		}
	}

	// 缓存结果
	s.setTagCache(cacheKey, tags)

	return tags, nil
}

// GetVirtualFoldersByTagID 通过标签获取虚拟文件夹
func (s *TagService) GetVirtualFoldersByTagID(ctx context.Context, userID uint, tagID uint) ([]model.VirtualFolder, error) {
	// 检查标签是否存在
	tag, err := s.TagRepo.GetTagByID(ctx, tagID)
	if err != nil {
		return nil, err
	}

	// 检查用户是否为项目成员
	isMember, err := s.ProjectRepo.IsProjectMember(ctx, tag.ProjectId, userID)
	if err != nil || !isMember {
		return nil, nil // 权限不足
	}

	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("virtual_folders_by_tag:%d", tagID)
	if cached, found := s.getVirtualFolderFromCache(cacheKey); found {
		return cached, nil
	}

	// 获取标签关系
	relations, err := s.TagRelationRepo.GetTagRelationsByTagID(ctx, tagID)
	if err != nil {
		return nil, err
	}

	// 提取虚拟文件夹ID
	var folderIDs []uint
	for _, relation := range relations {
		folderIDs = append(folderIDs, relation.VirtualFolderID)
	}

	// 批量获取虚拟文件夹
	var virtualFolders []model.VirtualFolder
	if len(folderIDs) > 0 {
		virtualFolders, err = s.VirtualFolderRepo.GetVirtualFoldersByIDs(ctx, folderIDs)
		if err != nil {
			return nil, err
		}
	}

	// 缓存结果
	s.setVirtualFolderCache(cacheKey, virtualFolders)

	return virtualFolders, nil
}

// 从标签缓存获取数据
func (s *TagService) getTagFromCache(key string) ([]model.Tag, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if data, found := s.tagCache[key]; found {
		return data, true
	}
	return nil, false
}

// 设置标签缓存
func (s *TagService) setTagCache(key string, data []model.Tag) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.tagCache[key] = data

	// 异步清除过期缓存
	go func() {
		time.Sleep(s.cacheExpiry)
		s.cacheMutex.Lock()
		delete(s.tagCache, key)
		s.cacheMutex.Unlock()
	}()
}

// 从虚拟文件夹缓存获取数据
func (s *TagService) getVirtualFolderFromCache(key string) ([]model.VirtualFolder, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if data, found := s.virtualFolderCache[key]; found {
		return data, true
	}
	return nil, false
}

// 设置虚拟文件夹缓存
func (s *TagService) setVirtualFolderCache(key string, data []model.VirtualFolder) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.virtualFolderCache[key] = data

	// 异步清除过期缓存
	go func() {
		time.Sleep(s.cacheExpiry)
		s.cacheMutex.Lock()
		delete(s.virtualFolderCache, key)
		s.cacheMutex.Unlock()
	}()
}

// 清除缓存
func (s *TagService) clearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.tagCache = make(map[string][]model.Tag)
	s.virtualFolderCache = make(map[string][]model.VirtualFolder)
}
