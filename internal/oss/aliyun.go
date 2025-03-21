package oss

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/myysophia/ossmanager-backend/internal/config"
	"github.com/myysophia/ossmanager-backend/internal/logger"
	"go.uber.org/zap"
	"io"
	"path"
	"time"
)

// AliyunOSSService 阿里云OSS存储服务
type AliyunOSSService struct {
	client     *oss.Client
	bucket     *oss.Bucket
	config     *config.AliyunOSSConfig
	bucketName string
	uploadDir  string
}

// NewAliyunOSSService 创建阿里云OSS存储服务
func NewAliyunOSSService(cfg *config.AliyunOSSConfig) (*AliyunOSSService, error) {
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("初始化阿里云OSS客户端失败: %w", err)
	}

	bucket, err := client.Bucket(cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("获取阿里云OSS Bucket失败: %w", err)
	}

	service := &AliyunOSSService{
		client:     client,
		bucket:     bucket,
		config:     cfg,
		bucketName: cfg.Bucket,
		uploadDir:  cfg.UploadDir,
	}

	return service, nil
}

// GetName 获取存储服务名称
func (s *AliyunOSSService) GetName() string {
	return "阿里云OSS"
}

// GetType 获取存储服务类型
func (s *AliyunOSSService) GetType() string {
	return StorageTypeAliyunOSS
}

// getObjectKey 获取对象键
func (s *AliyunOSSService) getObjectKey(filename string) string {
	return path.Join(s.uploadDir, filename)
}

// Upload 上传文件
func (s *AliyunOSSService) Upload(file io.Reader, objectKey string) (string, error) {
	fullObjectKey := s.getObjectKey(objectKey)
	err := s.bucket.PutObject(fullObjectKey, file)
	if err != nil {
		logger.Error("阿里云OSS上传文件失败", zap.String("objectKey", fullObjectKey), zap.Error(err))
		return "", fmt.Errorf("上传文件到阿里云OSS失败: %w", err)
	}

	// 返回可访问的URL
	signedURL, err := s.bucket.SignURL(fullObjectKey, oss.HTTPGet, int64(s.config.GetOSSURLExpiration().Seconds()))
	if err != nil {
		logger.Error("生成阿里云OSS下载URL失败", zap.String("objectKey", fullObjectKey), zap.Error(err))
		return "", fmt.Errorf("生成阿里云OSS下载URL失败: %w", err)
	}

	return signedURL, nil
}

// InitMultipartUpload 初始化分片上传
func (s *AliyunOSSService) InitMultipartUpload(filename string) (string, []string, error) {
	objectKey := s.getObjectKey(filename)
	// 初始化分片上传
	imur, err := s.bucket.InitiateMultipartUpload(objectKey)
	if err != nil {
		logger.Error("初始化阿里云OSS分片上传失败", zap.String("filename", filename), zap.Error(err))
		return "", nil, fmt.Errorf("初始化阿里云OSS分片上传失败: %w", err)
	}

	// 这里返回uploadID，前端需要保存此ID用于后续的分片上传和完成操作
	// 真实场景中，我们还需要根据文件大小计算分片数量，并为每个分片生成上传URL
	// 这里仅作为示例，实际上应该由前端计算分片并请求签名URL
	return imur.UploadID, nil, nil
}

// CompleteMultipartUpload 完成分片上传
func (s *AliyunOSSService) CompleteMultipartUpload(uploadID string, parts []Part, objectKey string) (string, error) {
	fullObjectKey := s.getObjectKey(objectKey)

	// 将我们的Part结构转换为阿里云SDK的Part结构
	ossParts := make([]oss.UploadPart, len(parts))
	for i, part := range parts {
		ossParts[i] = oss.UploadPart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		}
	}

	// 完成分片上传
	_, err := s.bucket.CompleteMultipartUpload(oss.InitiateMultipartUploadResult{
		Key:      fullObjectKey,
		UploadID: uploadID,
	}, ossParts)

	if err != nil {
		logger.Error("完成阿里云OSS分片上传失败",
			zap.String("objectKey", fullObjectKey),
			zap.String("uploadID", uploadID),
			zap.Error(err))
		return "", fmt.Errorf("完成阿里云OSS分片上传失败: %w", err)
	}

	// 生成下载URL
	signedURL, err := s.bucket.SignURL(fullObjectKey, oss.HTTPGet, int64(s.config.GetOSSURLExpiration().Seconds()))
	if err != nil {
		logger.Error("生成阿里云OSS下载URL失败", zap.String("objectKey", fullObjectKey), zap.Error(err))
		return "", fmt.Errorf("生成阿里云OSS下载URL失败: %w", err)
	}

	return signedURL, nil
}

// AbortMultipartUpload 取消分片上传
func (s *AliyunOSSService) AbortMultipartUpload(uploadID string, objectKey string) error {
	fullObjectKey := s.getObjectKey(objectKey)

	// 取消分片上传
	err := s.bucket.AbortMultipartUpload(oss.InitiateMultipartUploadResult{
		Key:      fullObjectKey,
		UploadID: uploadID,
	})

	if err != nil {
		logger.Error("取消阿里云OSS分片上传失败",
			zap.String("objectKey", fullObjectKey),
			zap.String("uploadID", uploadID),
			zap.Error(err))
		return fmt.Errorf("取消阿里云OSS分片上传失败: %w", err)
	}

	return nil
}

// GenerateDownloadURL 生成下载URL
func (s *AliyunOSSService) GenerateDownloadURL(objectKey string, expiration time.Duration) (string, time.Time, error) {
	fullObjectKey := s.getObjectKey(objectKey)

	// 设置过期时间
	expires := time.Now().Add(expiration)
	expiresSeconds := int64(expiration.Seconds())

	// 生成签名URL
	signedURL, err := s.bucket.SignURL(fullObjectKey, oss.HTTPGet, expiresSeconds)
	if err != nil {
		logger.Error("生成阿里云OSS下载URL失败", zap.String("objectKey", fullObjectKey), zap.Error(err))
		return "", time.Time{}, fmt.Errorf("生成阿里云OSS下载URL失败: %w", err)
	}

	return signedURL, expires, nil
}

// DeleteObject 删除对象
func (s *AliyunOSSService) DeleteObject(objectKey string) error {
	fullObjectKey := s.getObjectKey(objectKey)

	// 删除对象
	err := s.bucket.DeleteObject(fullObjectKey)
	if err != nil {
		logger.Error("删除阿里云OSS对象失败", zap.String("objectKey", fullObjectKey), zap.Error(err))
		return fmt.Errorf("删除阿里云OSS对象失败: %w", err)
	}

	return nil
}

// GetObjectInfo 获取对象信息
func (s *AliyunOSSService) GetObjectInfo(objectKey string) (int64, error) {
	fullObjectKey := s.getObjectKey(objectKey)

	// 获取对象元数据
	props, err := s.bucket.GetObjectDetailedMeta(fullObjectKey)
	if err != nil {
		logger.Error("获取阿里云OSS对象信息失败", zap.String("objectKey", fullObjectKey), zap.Error(err))
		return 0, fmt.Errorf("获取阿里云OSS对象信息失败: %w", err)
	}

	// 获取文件大小
	contentLength := props.Get("Content-Length")
	if contentLength == "" {
		return 0, fmt.Errorf("获取阿里云OSS对象大小失败: Content-Length为空")
	}

	var size int64
	_, err = fmt.Sscanf(contentLength, "%d", &size)
	if err != nil {
		return 0, fmt.Errorf("解析阿里云OSS对象大小失败: %w", err)
	}

	return size, nil
}

// GetBucketName 获取存储桶名称
func (s *AliyunOSSService) GetBucketName() string {
	return s.bucketName
}

// GetObject 获取对象内容
func (s *AliyunOSSService) GetObject(objectKey string) (io.ReadCloser, error) {
	fullObjectKey := s.getObjectKey(objectKey)
	body, err := s.bucket.GetObject(fullObjectKey)
	if err != nil {
		logger.Error("获取阿里云OSS对象失败", zap.String("objectKey", fullObjectKey), zap.Error(err))
		return nil, fmt.Errorf("获取阿里云OSS对象失败: %w", err)
	}
	return body, nil
}

// TriggerMD5Calculation 触发计算MD5值
func (s *AliyunOSSService) TriggerMD5Calculation(objectKey string, fileID uint) error {
	logger.Info("触发阿里云OSS对象MD5计算",
		zap.String("objectKey", objectKey),
		zap.Uint("fileID", fileID),
		zap.String("bucket", s.bucketName))

	// 此方法会在后台使用函数计算服务异步计算MD5，
	// 具体实现将在函数计算服务模块中处理
	if s.config.FunctionCompute.Enabled {
		// 后续会集成函数计算客户端来触发MD5计算
		// 目前的实现只是记录日志，表明功能已被触发
		logger.Info("阿里云OSS函数计算将异步计算MD5值")
		return nil
	} else {
		logger.Warn("阿里云OSS函数计算未启用，无法异步计算MD5值")
		return fmt.Errorf("阿里云OSS函数计算未启用，无法异步计算MD5值")
	}
}
