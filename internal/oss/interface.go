package oss

import (
	"io"
	"time"
)

// 存储类型枚举
const (
	StorageTypeAliyunOSS = "ALIYUN_OSS"
	StorageTypeAWSS3     = "AWS_S3"
	StorageTypeR2        = "CLOUDFLARE_R2"
)

// Part 分片信息
type Part struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

// StorageService 存储服务接口
type StorageService interface {
	// GetName 获取存储服务名称
	GetName() string

	// GetType 获取存储服务类型
	GetType() string

	// GetBucketName 获取存储桶名称
	GetBucketName() string

	// Upload 上传文件
	// file: 文件内容
	// objectKey: 对象键
	// 返回：对象URL和错误
	Upload(file io.Reader, objectKey string) (string, error)

	// InitMultipartUpload 初始化分片上传
	// filename: 文件名
	// 返回：上传ID, 上传URL列表, 错误
	InitMultipartUpload(filename string) (string, []string, error)

	// CompleteMultipartUpload 完成分片上传
	// uploadID: 上传ID
	// parts: 分片信息
	// objectKey: 对象键
	// 返回：对象URL, 错误
	CompleteMultipartUpload(uploadID string, parts []Part, objectKey string) (string, error)

	// AbortMultipartUpload 取消分片上传
	// uploadID: 上传ID
	// objectKey: 对象键
	AbortMultipartUpload(uploadID string, objectKey string) error

	// GenerateDownloadURL 生成下载URL
	// objectKey: 对象键
	// expiration: 过期时间
	// 返回：下载URL, 过期时间, 错误
	GenerateDownloadURL(objectKey string, expiration time.Duration) (string, time.Time, error)

	// DeleteObject 删除对象
	// objectKey: 对象键
	DeleteObject(objectKey string) error

	// GetObjectInfo 获取对象信息
	// objectKey: 对象键
	// 返回：对象大小, 错误
	GetObjectInfo(objectKey string) (int64, error)

	// GetObject 获取对象内容
	// objectKey: 对象键
	// 返回：对象内容读取器, 错误
	GetObject(objectKey string) (io.ReadCloser, error)

	// TriggerMD5Calculation 触发计算MD5值
	// objectKey: 对象键
	// fileID: 文件ID
	// 返回：错误
	TriggerMD5Calculation(objectKey string, fileID uint) error
}

// StorageFactory 存储服务工厂
type StorageFactory interface {
	// GetStorageService 获取存储服务
	// storageType: 存储类型
	GetStorageService(storageType string) (StorageService, error)

	// GetDefaultStorageService 获取默认存储服务
	GetDefaultStorageService() (StorageService, error)

	// ClearCache 清除缓存
	ClearCache()
}
