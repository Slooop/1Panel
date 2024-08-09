package dto

import (
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/model"
)

type BackupOperate struct {
	Operate string                `json:"operate" validate:"required,oneof=add remove update"`
	Data    []model.BackupAccount `json:"data" validate:"required"`
}

type BackupInfo struct {
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	Type       string    `json:"type"`
	Bucket     string    `json:"bucket"`
	BackupPath string    `json:"backupPath"`
	Vars       string    `json:"vars"`
}

type BackupSearchFile struct {
	Type string `json:"type" validate:"required"`
}

type CommonBackup struct {
	Type       string `json:"type" validate:"required,oneof=app mysql mariadb redis website postgresql"`
	Name       string `json:"name"`
	DetailName string `json:"detailName"`
	Secret     string `json:"secret"`
}
type CommonRecover struct {
	Source     string `json:"source" validate:"required,oneof=OSS S3 SFTP MINIO LOCAL COS KODO OneDrive WebDAV"`
	Type       string `json:"type" validate:"required,oneof=app mysql mariadb redis website postgresql"`
	Name       string `json:"name"`
	DetailName string `json:"detailName"`
	File       string `json:"file"`
	Secret     string `json:"secret"`
}

type RecordSearch struct {
	PageInfo
	Type       string `json:"type" validate:"required"`
	Name       string `json:"name"`
	DetailName string `json:"detailName"`
}

type RecordSearchByCronjob struct {
	PageInfo
	CronjobID uint `json:"cronjobID" validate:"required"`
}

type BackupRecords struct {
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	Source     string    `json:"source"`
	BackupType string    `json:"backupType"`
	FileDir    string    `json:"fileDir"`
	FileName   string    `json:"fileName"`
	Size       int64     `json:"size"`
}

type DownloadRecord struct {
	Source   string `json:"source" validate:"required,oneof=OSS S3 SFTP MINIO LOCAL COS KODO OneDrive WebDAV"`
	FileDir  string `json:"fileDir" validate:"required"`
	FileName string `json:"fileName" validate:"required"`
}