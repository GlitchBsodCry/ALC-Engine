package model

import (
	
)

type UpdateReason string

type FatherType string

const (
	UpdateReasonQuery  UpdateReason = "query"  // 查询导致的热点数据推送
	UpdateReasonCreate UpdateReason = "create" // 新建实体
	UpdateReasonModify UpdateReason = "modify" // 实体信息被修改
	UpdateReasonDelete UpdateReason = "delete" // 实体被删除
)

const (
	FatherTypeRoot          FatherType = "root"
	FatherTypeVirtualFolder FatherType = "virtual_folder"
)


type CacheUpdateMessage struct {
	ProjectID       int64        `json:"project_id"`
	VirtualFolderID int64        `json:"virtual_folder_id"`
	FatherType      FatherType   `json:"father_type"`
	FatherID        int64        `json:"father_id"`
	Reason          UpdateReason `json:"reason"`
	Name            string       `json:"name"`
	Files           []FileInfo   `json:"files"`
}

type FileInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
