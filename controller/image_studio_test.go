package controller

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func imageStudioTestContext(method, path string, body *bytes.Buffer, userID, role int) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	if body == nil {
		body = bytes.NewBuffer(nil)
	}
	ctx.Request = httptest.NewRequest(method, path, body)
	ctx.Set("id", userID)
	ctx.Set("role", role)
	return ctx, recorder
}

func TestImageStudioIndependentLegalAndPrivateAssets(t *testing.T) {
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open("file:image-studio-test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	t.Cleanup(func() {
		connection, _ := db.DB()
		_ = connection.Close()
		model.DB = previousDB
	})
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, db.Create(&model.User{Username: "retained-user"}).Error)
	for range 2 {
		require.NoError(t, db.AutoMigrate(&model.ImageStudioConfig{}, &model.ImageStudioRecord{}, &model.ImageStudioAsset{}))
	}
	var user model.User
	require.NoError(t, db.Where("username = ?", "retained-user").First(&user).Error)
	assert.Equal(t, "retained-user", user.Username)

	defaults, err := model.GetImageStudioConfig()
	require.NoError(t, err)
	assert.Contains(t, defaults.Agreement, "本协议仅适用于本站控制台的在线生图功能")
	assert.Contains(t, defaults.Privacy, "不替代或修改本站其他服务的隐私政策")
	assert.NotEqual(t, model.ImageStudioRevision(defaults.Agreement), model.ImageStudioRevision(defaults.Privacy))

	defaults.OperatorName, defaults.ContactEmail = "Example", "contact@example.org"
	defaults.LocalDir = t.TempDir()
	require.NoError(t, db.Save(&defaults).Error)
	ctx, output := imageStudioTestContext(http.MethodGet, "/api/image-studio/legal/agreement", nil, 7, common.RoleCommonUser)
	ctx.Params = gin.Params{{Key: "kind", Value: "agreement"}}
	GetImageStudioLegal(ctx)
	assert.Equal(t, http.StatusOK, output.Code)
	assert.Contains(t, output.Body.String(), "contact@example.org")
	assert.NotContains(t, output.Body.String(), defaults.Privacy)

	form := &bytes.Buffer{}
	writer := multipart.NewWriter(form)
	for key, value := range map[string]string{
		"model": "model", "group": "default", "prompt": "landscape",
		"consent": "true", "agreement_revision": "stale",
		"privacy_revision": model.ImageStudioRevision(defaults.Privacy),
	} {
		require.NoError(t, writer.WriteField(key, value))
	}
	require.NoError(t, writer.Close())
	ctx, output = imageStudioTestContext(http.MethodPost, "/api/image-studio/generate", form, 7, common.RoleCommonUser)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	ImageStudioPrepare(ctx)
	assert.Equal(t, http.StatusConflict, output.Code)
	assert.Contains(t, output.Body.String(), "current image studio terms")

	record := model.ImageStudioRecord{
		UserID: 7, Prompt: "private prompt", Error: "private upstream detail", Status: "completed",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, db.Create(&record).Error)
	asset, err := service.StoreImageStudioAsset(t.Context(), defaults, record.ID, "result", "image/png", []byte("private bytes"))
	require.NoError(t, err)

	path := fmt.Sprintf("/api/image-studio/records/%d/assets/%d", record.ID, asset.ID)
	ctx, output = imageStudioTestContext(http.MethodGet, path, nil, 8, common.RoleCommonUser)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(record.ID)}, {Key: "asset", Value: fmt.Sprint(asset.ID)}}
	GetImageStudioAsset(ctx)
	assert.Equal(t, http.StatusNotFound, output.Code)
	assert.NotContains(t, output.Body.String(), "private bytes")

	ctx, output = imageStudioTestContext(http.MethodGet, path, nil, 7, common.RoleCommonUser)
	ctx.Params = gin.Params{{Key: "id", Value: "1 OR 1=1"}, {Key: "asset", Value: fmt.Sprint(asset.ID)}}
	GetImageStudioAsset(ctx)
	assert.Equal(t, http.StatusNotFound, output.Code)

	ctx, output = imageStudioTestContext(http.MethodGet, path, nil, 7, common.RoleCommonUser)
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprint(record.ID)}, {Key: "asset", Value: fmt.Sprint(asset.ID)}}
	GetImageStudioAsset(ctx)
	assert.Equal(t, http.StatusOK, output.Code)
	assert.Equal(t, "private bytes", output.Body.String())

	require.NoError(t, service.PurgeImageStudioRecord(t.Context(), &record, false))
	require.NoError(t, db.First(&record, record.ID).Error)
	assert.Empty(t, record.Prompt)
	assert.Empty(t, record.Error)
	assert.Equal(t, "expired", record.Status)
	var remaining int64
	require.NoError(t, db.Model(&model.ImageStudioAsset{}).Where("id = ?", asset.ID).Count(&remaining).Error)
	assert.Zero(t, remaining)
}

func TestImageStudioS3AssetSurvivesBackendSwitch(t *testing.T) {
	previousDB := model.DB
	db, err := gorm.Open(sqlite.Open("file:image-studio-s3-test?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	t.Cleanup(func() {
		connection, _ := db.DB()
		_ = connection.Close()
		model.DB = previousDB
	})
	require.NoError(t, db.AutoMigrate(&model.ImageStudioAsset{}))
	var stored []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/bucket/42-", r.URL.Path[:min(len(r.URL.Path), len("/bucket/42-"))])
		assert.Contains(t, r.Header.Get("Authorization"), "AWS4-HMAC-SHA256")
		switch r.Method {
		case http.MethodPut:
			stored, _ = io.ReadAll(r.Body)
		case http.MethodGet:
			_, _ = w.Write(stored)
		case http.MethodDelete:
			stored = nil
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	config := model.DefaultImageStudioConfig()
	config.StorageMode, config.S3Endpoint, config.S3Bucket, config.S3Region = "s3", server.URL, "bucket", "test-region"
	config.S3AccessKey, config.S3SecretKey = "key", "secret"
	asset, err := service.StoreImageStudioAsset(t.Context(), config, 42, "result", "image/png", []byte("image"))
	require.NoError(t, err)
	config.StorageMode = "local"
	data, err := service.ReadImageStudioAsset(t.Context(), asset)
	require.NoError(t, err)
	assert.Equal(t, "image", string(data))
	require.NoError(t, service.DeleteImageStudioAsset(t.Context(), asset))
	assert.Empty(t, stored)
}

func TestImageStudioCaptureIsolatesUpstreamHeaders(t *testing.T) {
	writer := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(writer)
	capture := &imageStudioCapture{ResponseWriter: ctx.Writer, status: http.StatusOK, header: ctx.Writer.Header().Clone()}
	capture.Header().Set("Content-Length", "999999")
	capture.WriteHeader(http.StatusCreated)
	_, err := capture.WriteString(`{"data":[]}`)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, capture.Status())
	assert.Equal(t, `{"data":[]}`, capture.body.String())
	assert.Empty(t, ctx.Writer.Header().Get("Content-Length"))
	assert.Empty(t, writer.Body.String())
}
