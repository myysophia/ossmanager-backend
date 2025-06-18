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

	// Upload 上传文件到默认存储桶
	Upload(file io.Reader, objectKey string) (string, error)

	// UploadToBucket 上传文件到指定的存储桶
	UploadToBucket(file io.Reader, objectKey string, regionCode string, bucketName string) (string, error)

	// InitMultipartUpload 初始化分片上传
	InitMultipartUpload(objectKey string) (string, []string, error)

	// InitMultipartUploadToBucket 初始化分片上传到指定的存储桶
	InitMultipartUploadToBucket(objectKey string, regionCode string, bucketName string) (string, []string, error)

	// CompleteMultipartUpload 完成分片上传
	CompleteMultipartUpload(objectKey string, uploadID string, parts []Part) (string, error)

	// CompleteMultipartUploadToBucket 完成分片上传到指定的存储桶
	CompleteMultipartUploadToBucket(objectKey string, uploadID string, parts []Part, regionCode string, bucketName string) (string, error)

	// GenerateUploadURL 生成上传URL
	// objectKey: 对象键
	// regionCode: 区域代码
	// bucketName: 存储桶名称
	GenerateUploadURL(objectKey, regionCode, bucketName string) (string, error)

	// AbortMultipartUpload 取消分片上传
	AbortMultipartUpload(objectKey string, uploadID string) error

	// GenerateDownloadURL 生成下载URL
	// objectKey: 对象键
	// expiration: 过期时间
	// 返回：下载URL, 过期时间, 错误
	GenerateDownloadURL(objectKey string, expiration time.Duration) (string, time.Time, error)

	// DeleteObject 删除文件
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

	// GetDownloadURL 获取文件下载URL
	GetDownloadURL(objectKey string, expires time.Duration) (string, error)
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
