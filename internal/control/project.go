package control

import (
	"mygo_bangforai/api/errors"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var ProjectService *service.ProjectService
var VirtualFolderService *service.VirtualFolderService
var FileService *service.FileService
var MountRelationService *service.MountRelationService
var TagService *service.TagService
var CallRelationService *service.CallRelationService

func InitProjectService(svc *service.ProjectService) {
	ProjectService = svc
}

func InitVirtualFolderService(svc *service.VirtualFolderService) {
	VirtualFolderService = svc
}

func InitFileService(svc *service.FileService) {
	FileService = svc
}

func InitMountRelationService(svc *service.MountRelationService) {
	MountRelationService = svc
}

func InitTagService(svc *service.TagService) {
	TagService = svc
}

func InitCallRelationService(svc *service.CallRelationService) {
	CallRelationService = svc
}

func CreateProject(c *gin.Context) {
	var req model.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id") // 从请求头中获取用户ID

	ctx := c.Request.Context()
	err := ProjectService.CreateProjectService(ctx, userID, req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}
	logger.Info("项目创建成功", zap.String("name", req.Name), zap.Uint("owner_id", userID))
	errors.Success(c, gin.H{
		"name":    req.Name,
		"message": "项目创建成功",
	})
}

func GetProjectList(c *gin.Context) {
	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	projects, err := ProjectService.GetProjectListService(ctx, userID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"projects": projects,
		"total":    len(projects),
	})
}

func GetProjectMembers(c *gin.Context) {

	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		errors.ParamError(c, "项目ID不能为空")
		return
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		errors.ParamError(c, "项目ID无效")
		return
	}

	ctx := c.Request.Context()
	members, err := ProjectService.GetProjectMembersService(ctx, uint(projectID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"members": members,
		"total":   len(members),
	})
}

func AddProjectMember(c *gin.Context) {
	var req model.AddProjectMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := ProjectService.AddProjectMemberService(ctx, userID, req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "项目成员添加成功",
	})
}

func UpdateProjectMember(c *gin.Context) {
	var req model.UpdateProjectMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := ProjectService.UpdateProjectMemberRoleService(ctx, userID, req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "项目成员角色更新成功",
	})
}

// CreateVirtualFolder 创建虚拟文件夹
func CreateVirtualFolder(c *gin.Context) {
	var req struct {
		RootID   uint   `json:"root_id" binding:"required"`
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := VirtualFolderService.CreateVirtualFolder(ctx, userID, req.RootID, req.Name, req.ParentID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "虚拟文件夹创建成功",
	})
}

// GetVirtualFoldersByRootID 查询根目录下的虚拟文件夹
func GetVirtualFoldersByRootID(c *gin.Context) {
	rootIDStr := c.Param("root_id")
	rootID, err := strconv.ParseUint(rootIDStr, 10, 32)
	if err != nil || rootID == 0 {
		errors.ParamError(c, "根目录ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	folders, err := VirtualFolderService.GetVirtualFoldersByRootID(ctx, userID, uint(rootID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"folders": folders,
		"total":   len(folders),
	})
}

// GetVirtualFoldersByParentID 查询父文件夹下的虚拟文件夹
func GetVirtualFoldersByParentID(c *gin.Context) {
	parentIDStr := c.Param("parent_id")
	parentID, err := strconv.ParseUint(parentIDStr, 10, 32)
	if err != nil || parentID == 0 {
		errors.ParamError(c, "父文件夹ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	folders, err := VirtualFolderService.GetVirtualFoldersByParentID(ctx, userID, uint(parentID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"folders": folders,
		"total":   len(folders),
	})
}

// UpdateVirtualFolder 修改虚拟文件夹
func UpdateVirtualFolder(c *gin.Context) {
	var req struct {
		FolderID uint   `json:"folder_id" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := VirtualFolderService.UpdateVirtualFolder(ctx, userID, req.FolderID, req.Name)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "虚拟文件夹修改成功",
	})
}

// DeleteVirtualFolder 删除虚拟文件夹
func DeleteVirtualFolder(c *gin.Context) {
	folderIDStr := c.Param("folder_id")
	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil || folderID == 0 {
		errors.ParamError(c, "文件夹ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err = VirtualFolderService.DeleteVirtualFolder(ctx, userID, uint(folderID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "虚拟文件夹删除成功",
	})
}

// MoveVirtualFolder 移动虚拟文件夹
func MoveVirtualFolder(c *gin.Context) {
	var req struct {
		FolderID      uint   `json:"folder_id" binding:"required"`
		NewParentID   uint   `json:"new_parent_id" binding:"required"`
		NewParentType string `json:"new_parent_type" binding:"required,oneof=root folder"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := VirtualFolderService.MoveVirtualFolder(ctx, userID, req.FolderID, req.NewParentID, req.NewParentType)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "虚拟文件夹移动成功",
	})
}

// MountFile 将文件挂载到虚拟文件夹
func MountFile(c *gin.Context) {
	var req struct {
		ParentID   uint   `json:"parent_id" binding:"required"`
		ParentType string `json:"parent_type" binding:"required,oneof=root folder"`
		ChildID    uint   `json:"child_id" binding:"required"`
		ChildType  string `json:"child_type" binding:"required,oneof=cloud real"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := FileService.MountFile(ctx, userID, req.ParentID, req.ParentType, req.ChildID, req.ChildType)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件挂载成功",
	})
}

// UploadFileToCloud 上传文件到云端
func UploadFileToCloud(c *gin.Context) {
	var req struct {
		RootID uint   `json:"root_id" binding:"required"`
		Name   string `json:"name" binding:"required"`
		Path   string `json:"path" binding:"required"`
		Hash   string `json:"hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	fileID, err := FileService.UploadFileToCloud(ctx, userID, req.RootID, req.Name, req.Path, req.Hash)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件上传成功",
		"file_id": fileID,
	})
}

// DownloadCloudFileToLocal 下载云文件到本地
func DownloadCloudFileToLocal(c *gin.Context) {
	var req struct {
		CloudFileID uint   `json:"cloud_file_id" binding:"required"`
		LocalPath   string `json:"local_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := FileService.DownloadCloudFileToLocal(ctx, userID, req.CloudFileID, req.LocalPath)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件下载成功",
	})
}

// RenameFile 修改文件名
func RenameFile(c *gin.Context) {
	var req struct {
		FileID   uint   `json:"file_id" binding:"required"`
		FileType string `json:"file_type" binding:"required,oneof=cloud real"`
		NewName  string `json:"new_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := FileService.RenameFile(ctx, userID, req.FileID, req.FileType, req.NewName)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件名修改成功",
	})
}

// GetProjectEvents 获取项目实时事件（SSE）
func GetProjectEvents(c *gin.Context) {
	// 这是一个占位函数，实际实现需要集成SSE功能
	// 目前返回空数据，表示功能待实现
	errors.Success(c, gin.H{
		"events":  []interface{}{},
		"message": "实时事件功能待实现",
	})
}

// ApproveChange 批准变更请求
func ApproveChange(c *gin.Context) {
	// 这是一个占位函数，实际实现需要集成变更审批功能
	// 目前返回成功消息，表示功能待实现
	errors.Success(c, gin.H{
		"message": "变更审批功能待实现",
	})
}

// DeleteFile 删除文件（不删除真实文件）
func DeleteFile(c *gin.Context) {
	var req struct {
		FileID   uint   `json:"file_id" binding:"required"`
		FileType string `json:"file_type" binding:"required,oneof=cloud real"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := FileService.DeleteFile(ctx, userID, req.FileID, req.FileType)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件删除成功",
	})
}

// MoveFile 移动文件
func MoveFile(c *gin.Context) {
	var req struct {
		FileID        uint   `json:"file_id" binding:"required"`
		FileType      string `json:"file_type" binding:"required,oneof=cloud real"`
		NewParentID   uint   `json:"new_parent_id" binding:"required"`
		NewParentType string `json:"new_parent_type" binding:"required,oneof=root folder"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := FileService.MoveFile(ctx, userID, req.FileID, req.FileType, req.NewParentID, req.NewParentType)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件移动成功",
	})
}

// CopyFile 复制文件
func CopyFile(c *gin.Context) {
	var req struct {
		FileID        uint   `json:"file_id" binding:"required"`
		FileType      string `json:"file_type" binding:"required,oneof=cloud real"`
		NewParentID   uint   `json:"new_parent_id" binding:"required"`
		NewParentType string `json:"new_parent_type" binding:"required,oneof=root folder"`
		NewName       string `json:"new_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := FileService.CopyFile(ctx, userID, req.FileID, req.FileType, req.NewParentID, req.NewParentType, req.NewName)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "文件复制成功",
	})
}

// GetFilesByFolderID 获取文件夹下的文件列表
func GetFilesByFolderID(c *gin.Context) {
	folderIDStr := c.Param("folder_id")
	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil || folderID == 0 {
		errors.ParamError(c, "文件夹ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	files, err := FileService.GetFilesByFolderID(ctx, userID, uint(folderID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"files": files,
		"total": len(files),
	})
}

// GetFileCache 获取文件缓存
func GetFileCache(c *gin.Context) {
	cloudFileIDStr := c.Param("cloud_file_id")
	cloudFileID, err := strconv.ParseUint(cloudFileIDStr, 10, 32)
	if err != nil || cloudFileID == 0 {
		errors.ParamError(c, "云文件ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	cache, err := FileService.GetFileCache(ctx, userID, uint(cloudFileID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"cache": cache,
	})
}

// CreateMountRelation 创建挂载关系
func CreateMountRelation(c *gin.Context) {
	var req struct {
		ParentID     uint   `json:"parent_id" binding:"required"`
		ParentType   string `json:"parent_type" binding:"required,oneof=root folder"`
		ChildID      uint   `json:"child_id" binding:"required"`
		ChildType    string `json:"child_type" binding:"required,oneof=folder cloud real"`
		RelationType string `json:"relation_type" binding:"required,oneof=mount call"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := MountRelationService.CreateMountRelation(ctx, userID, req.ParentID, req.ParentType, req.ChildID, req.ChildType, req.RelationType)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "挂载关系创建成功",
	})
}

// GetMountRelationsByParentID 根据父节点ID和类型获取挂载关系
func GetMountRelationsByParentID(c *gin.Context) {
	parentIDStr := c.Param("parent_id")
	parentID, err := strconv.ParseUint(parentIDStr, 10, 32)
	if err != nil || parentID == 0 {
		errors.ParamError(c, "父节点ID不能为空或无效")
		return
	}

	parentType := c.Query("parent_type")
	if parentType != "root" && parentType != "folder" {
		errors.ParamError(c, "父节点类型必须为root或folder")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	relations, err := MountRelationService.GetMountRelationsByParentID(ctx, userID, uint(parentID), parentType)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"relations": relations,
		"total":     len(relations),
	})
}

// GetMountRelationsByChildID 根据子节点ID和类型获取挂载关系
func GetMountRelationsByChildID(c *gin.Context) {
	childIDStr := c.Param("child_id")
	childID, err := strconv.ParseUint(childIDStr, 10, 32)
	if err != nil || childID == 0 {
		errors.ParamError(c, "子节点ID不能为空或无效")
		return
	}

	childType := c.Query("child_type")
	if childType != "folder" && childType != "cloud" && childType != "real" {
		errors.ParamError(c, "子节点类型必须为folder、cloud或real")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	relations, err := MountRelationService.GetMountRelationsByChildID(ctx, userID, uint(childID), childType)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"relations": relations,
		"total":     len(relations),
	})
}

// UpdateMountRelation 更新挂载关系
func UpdateMountRelation(c *gin.Context) {
	var req model.MountRelation
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := MountRelationService.UpdateMountRelation(ctx, userID, &req)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "挂载关系更新成功",
	})
}

// DeleteMountRelation 删除挂载关系
func DeleteMountRelation(c *gin.Context) {
	var req struct {
		ParentID uint `json:"parent_id" binding:"required"`
		ChildID  uint `json:"child_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := MountRelationService.DeleteMountRelation(ctx, userID, req.ParentID, req.ChildID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "挂载关系删除成功",
	})
}

// CreateTag 创建标签
func CreateTag(c *gin.Context) {
	var req struct {
		ProjectID uint   `json:"project_id" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Color     string `json:"color" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := TagService.CreateTag(ctx, userID, req.ProjectID, req.Name, req.Color)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "标签创建成功",
	})
}

// GetTagsByProjectID 获取项目的所有标签
func GetTagsByProjectID(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil || projectID == 0 {
		errors.ParamError(c, "项目ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	tags, err := TagService.GetTagsByProjectID(ctx, userID, uint(projectID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"tags":  tags,
		"total": len(tags),
	})
}

// UpdateTag 更新标签
func UpdateTag(c *gin.Context) {
	var req struct {
		TagID uint   `json:"tag_id" binding:"required"`
		Name  string `json:"name" binding:"required"`
		Color string `json:"color" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := TagService.UpdateTag(ctx, userID, req.TagID, req.Name, req.Color)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "标签更新成功",
	})
}

// DeleteTag 删除标签
func DeleteTag(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 32)
	if err != nil || tagID == 0 {
		errors.ParamError(c, "标签ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err = TagService.DeleteTag(ctx, userID, uint(tagID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "标签删除成功",
	})
}

// AddTagToVirtualFolder 为虚拟文件夹添加标签
func AddTagToVirtualFolder(c *gin.Context) {
	var req struct {
		TagID           uint `json:"tag_id" binding:"required"`
		VirtualFolderID uint `json:"virtual_folder_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := TagService.AddTagToVirtualFolder(ctx, userID, req.TagID, req.VirtualFolderID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "标签添加成功",
	})
}

// RemoveTagFromVirtualFolder 从虚拟文件夹移除标签
func RemoveTagFromVirtualFolder(c *gin.Context) {
	var req struct {
		TagID           uint `json:"tag_id" binding:"required"`
		VirtualFolderID uint `json:"virtual_folder_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err := TagService.RemoveTagFromVirtualFolder(ctx, userID, req.TagID, req.VirtualFolderID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message": "标签移除成功",
	})
}

// GetTagsByVirtualFolderID 获取虚拟文件夹的标签
func GetTagsByVirtualFolderID(c *gin.Context) {
	virtualFolderIDStr := c.Param("virtual_folder_id")
	virtualFolderID, err := strconv.ParseUint(virtualFolderIDStr, 10, 32)
	if err != nil || virtualFolderID == 0 {
		errors.ParamError(c, "虚拟文件夹ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	tags, err := TagService.GetTagsByVirtualFolderID(ctx, userID, uint(virtualFolderID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"tags":  tags,
		"total": len(tags),
	})
}

// GetVirtualFoldersByTagID 通过标签获取虚拟文件夹
func GetVirtualFoldersByTagID(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 32)
	if err != nil || tagID == 0 {
		errors.ParamError(c, "标签ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	virtualFolders, err := TagService.GetVirtualFoldersByTagID(ctx, userID, uint(tagID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"virtual_folders": virtualFolders,
		"total":           len(virtualFolders),
	})
}

// CreateCallRelation 创建虚拟文件夹调用关系
func CreateCallRelation(c *gin.Context) {
	var req struct {
		CallerFolderID uint `json:"caller_folder_id" binding:"required"`
		CalledFolderID uint `json:"called_folder_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.ParamError(c, err.Error())
		return
	}

	// 从路径参数获取项目ID
	projectIDStr := c.Param("project_id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil || projectID == 0 {
		errors.ParamError(c, "项目ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	err = CallRelationService.CreateCallRelation(ctx, uint(projectID), req.CallerFolderID, req.CalledFolderID, userID)
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"message":          "调用关系创建成功",
		"caller_folder_id": req.CallerFolderID,
		"called_folder_id": req.CalledFolderID,
	})
}

// GetFolderInfo 查询虚拟文件夹基本信息
func GetFolderInfo(c *gin.Context) {
	folderIDStr := c.Param("folder_id")
	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil || folderID == 0 {
		errors.ParamError(c, "文件夹ID不能为空或无效")
		return
	}

	userID := c.GetUint("user_id")

	ctx := c.Request.Context()
	folder, err := VirtualFolderService.GetVirtualFolderByID(ctx, userID, uint(folderID))
	if err != nil {
		errors.Error(c, errors.InternalError, err)
		return
	}

	errors.Success(c, gin.H{
		"folder":  folder,
		"message": "文件夹信息获取成功",
	})
}
