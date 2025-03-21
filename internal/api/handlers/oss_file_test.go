package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/myysophia/ossmanager-backend/internal/db/models"
	"github.com/myysophia/ossmanager-backend/internal/tests/mocks"
	"github.com/myysophia/ossmanager-backend/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestOSSFileHandler_Upload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockStorage := mocks.NewMockStorageService(ctrl)

	handler := NewOSSFileHandler(mockDB, mockStorage)

	// 设置测试数据
	file := &models.OSSFile{
		ID:            1,
		ConfigID:      1,
		OriginalName:  "test.txt",
		StorageName:   "test.txt",
		Size:          100,
		ContentType:   "text/plain",
		UploadStatus:  models.UploadStatusCompleted,
		UploaderID:    1,
		UploaderName:  "test",
		StorageConfig: &models.OSSConfig{},
	}

	// 设置 Mock 期望
	mockDB.EXPECT().First(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)
	mockStorage.EXPECT().Upload(gomock.Any()).Return("http://example.com/test.txt", nil)
	mockDB.EXPECT().Create(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))
	c.Set("username", "test")

	// 设置请求体
	body := map[string]interface{}{
		"config_id":     1,
		"original_name": "test.txt",
		"size":          100,
		"content_type":  "text/plain",
	}
	jsonBody, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/v1/files", bytes.NewBuffer(jsonBody))

	// 执行测试
	handler.Upload(c)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestOSSFileHandler_InitMultipartUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockStorage := mocks.NewMockStorageService(ctrl)

	handler := NewOSSFileHandler(mockDB, mockStorage)

	// 设置测试数据
	file := &models.OSSFile{
		ID:            1,
		ConfigID:      1,
		OriginalName:  "test.txt",
		StorageName:   "test.txt",
		Size:          100,
		ContentType:   "text/plain",
		UploadStatus:  models.UploadStatusInProgress,
		UploaderID:    1,
		UploaderName:  "test",
		StorageConfig: &models.OSSConfig{},
	}

	// 设置 Mock 期望
	mockDB.EXPECT().First(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)
	mockStorage.EXPECT().InitMultipartUpload(gomock.Any()).Return("upload-id", nil)
	mockDB.EXPECT().Create(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))
	c.Set("username", "test")

	// 设置请求体
	body := map[string]interface{}{
		"config_id":     1,
		"original_name": "test.txt",
		"size":          100,
		"content_type":  "text/plain",
	}
	jsonBody, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/v1/files/multipart", bytes.NewBuffer(jsonBody))

	// 执行测试
	handler.InitMultipartUpload(c)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestOSSFileHandler_CompleteMultipartUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockStorage := mocks.NewMockStorageService(ctrl)

	handler := NewOSSFileHandler(mockDB, mockStorage)

	// 设置测试数据
	file := &models.OSSFile{
		ID:            1,
		ConfigID:      1,
		OriginalName:  "test.txt",
		StorageName:   "test.txt",
		Size:          100,
		ContentType:   "text/plain",
		UploadStatus:  models.UploadStatusInProgress,
		UploaderID:    1,
		UploaderName:  "test",
		StorageConfig: &models.OSSConfig{},
	}

	// 设置 Mock 期望
	mockDB.EXPECT().First(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)
	mockStorage.EXPECT().CompleteMultipartUpload(gomock.Any()).Return("http://example.com/test.txt", nil)
	mockDB.EXPECT().Save(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))
	c.Set("username", "test")

	// 设置请求体
	body := map[string]interface{}{
		"file_id": 1,
	}
	jsonBody, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/v1/files/multipart/complete", bytes.NewBuffer(jsonBody))

	// 执行测试
	handler.CompleteMultipartUpload(c)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestOSSFileHandler_AbortMultipartUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockStorage := mocks.NewMockStorageService(ctrl)

	handler := NewOSSFileHandler(mockDB, mockStorage)

	// 设置测试数据
	file := &models.OSSFile{
		ID:            1,
		ConfigID:      1,
		OriginalName:  "test.txt",
		StorageName:   "test.txt",
		Size:          100,
		ContentType:   "text/plain",
		UploadStatus:  models.UploadStatusInProgress,
		UploaderID:    1,
		UploaderName:  "test",
		StorageConfig: &models.OSSConfig{},
	}

	// 设置 Mock 期望
	mockDB.EXPECT().First(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)
	mockStorage.EXPECT().AbortMultipartUpload(gomock.Any()).Return(nil)
	mockDB.EXPECT().Delete(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))
	c.Set("username", "test")

	// 设置请求体
	body := map[string]interface{}{
		"file_id": 1,
	}
	jsonBody, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/v1/files/multipart/abort", bytes.NewBuffer(jsonBody))

	// 执行测试
	handler.AbortMultipartUpload(c)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestOSSFileHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockStorage := mocks.NewMockStorageService(ctrl)

	handler := NewOSSFileHandler(mockDB, mockStorage)

	// 设置测试数据
	files := []models.OSSFile{
		{
			ID:            1,
			ConfigID:      1,
			OriginalName:  "test1.txt",
			StorageName:   "test1.txt",
			Size:          100,
			ContentType:   "text/plain",
			UploadStatus:  models.UploadStatusCompleted,
			UploaderID:    1,
			UploaderName:  "test",
			StorageConfig: &models.OSSConfig{},
		},
		{
			ID:            2,
			ConfigID:      1,
			OriginalName:  "test2.txt",
			StorageName:   "test2.txt",
			Size:          200,
			ContentType:   "text/plain",
			UploadStatus:  models.UploadStatusCompleted,
			UploaderID:    1,
			UploaderName:  "test",
			StorageConfig: &models.OSSConfig{},
		},
	}

	// 设置 Mock 期望
	mockDB.EXPECT().Where(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Preload(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Offset(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Limit(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Find(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)
	mockDB.EXPECT().Model(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Where(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Count(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))
	c.Set("username", "test")

	// 设置查询参数
	c.Request = httptest.NewRequest("GET", "/api/v1/files?config_id=1&page=1&page_size=10", nil)

	// 执行测试
	handler.List(c)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestOSSFileHandler_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockStorage := mocks.NewMockStorageService(ctrl)

	handler := NewOSSFileHandler(mockDB, mockStorage)

	// 设置测试数据
	file := &models.OSSFile{
		ID:            1,
		ConfigID:      1,
		OriginalName:  "test.txt",
		StorageName:   "test.txt",
		Size:          100,
		ContentType:   "text/plain",
		UploadStatus:  models.UploadStatusCompleted,
		UploaderID:    1,
		UploaderName:  "test",
		StorageConfig: &models.OSSConfig{},
	}

	// 设置 Mock 期望
	mockDB.EXPECT().First(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)
	mockStorage.EXPECT().DeleteObject(gomock.Any()).Return(nil)
	mockDB.EXPECT().Delete(gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))
	c.Set("username", "test")

	// 设置请求参数
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// 执行测试
	handler.Delete(c)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestOSSFileHandler_GetDownloadURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDB(ctrl)
	mockStorage := mocks.NewMockStorageService(ctrl)

	handler := NewOSSFileHandler(mockDB, mockStorage)

	// 设置测试数据
	file := &models.OSSFile{
		ID:            1,
		ConfigID:      1,
		OriginalName:  "test.txt",
		StorageName:   "test.txt",
		Size:          100,
		ContentType:   "text/plain",
		UploadStatus:  models.UploadStatusCompleted,
		UploaderID:    1,
		UploaderName:  "test",
		StorageConfig: &models.OSSConfig{},
	}

	// 设置 Mock 期望
	mockDB.EXPECT().First(gomock.Any(), gomock.Any()).Return(mockDB)
	mockDB.EXPECT().Error().Return(nil)
	mockStorage.EXPECT().GenerateDownloadURL(gomock.Any()).Return("http://example.com/test.txt", nil)

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))
	c.Set("username", "test")

	// 设置请求参数
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// 执行测试
	handler.GetDownloadURL(c)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp utils.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}
