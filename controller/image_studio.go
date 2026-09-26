package controller

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/mail"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type imageStudioModelInfo struct {
	Name       string             `json:"name"`
	AllowEdits bool               `json:"allow_edits"`
	Price      float64            `json:"price"`
	Groups     map[string]float64 `json:"groups"`
}

func imageStudioCatalog(config model.ImageStudioConfig, userID int, admin bool) []imageStudioModelInfo {
	group := ""
	if !admin {
		group, _ = model.GetUserGroup(userID, false)
	}
	allowed := service.GetUserUsableGroups(group)
	enabled := config.EnabledModels()
	result := make([]imageStudioModelInfo, 0)
	for _, item := range model.GetPricing() {
		currentPrice, fixedPrice := ratio_setting.GetModelPrice(item.ModelName, false)
		if billing_setting.GetBillingMode(item.ModelName) == billing_setting.BillingModeTieredExpr {
			expression, exists := billing_setting.GetBillingExpr(item.ModelName)
			currentPrice, fixedPrice = billingexpr.ImageUnitPrice(expression)
			fixedPrice = fixedPrice && exists
		}
		if !slices.Contains(item.SupportedEndpointTypes, constant.EndpointTypeImageGeneration) ||
			strings.HasPrefix(item.ModelName, "dall-e") ||
			!fixedPrice ||
			math.IsNaN(currentPrice) || math.IsInf(currentPrice, 0) || currentPrice < 0 {
			continue
		}
		choice := imageStudioModelInfo{Name: item.ModelName, Price: currentPrice, Groups: map[string]float64{}}
		if admin {
			choice.AllowEdits = slices.ContainsFunc(enabled, func(v model.ImageStudioModel) bool { return v.Name == item.ModelName && v.AllowEdits })
		} else {
			found := false
			for _, v := range enabled {
				if v.Name == item.ModelName {
					choice.AllowEdits, found = v.AllowEdits, true
					break
				}
			}
			if !found {
				continue
			}
		}
		if admin {
			for _, name := range item.EnableGroup {
				if name != "all" && name != "auto" && ratio_setting.ContainsGroupRatio(name) {
					choice.Groups[name] = service.GetUserGroupRatio(group, name)
				}
			}
		} else {
			for name := range allowed {
				if name != "auto" && ratio_setting.ContainsGroupRatio(name) &&
					slices.Contains(service.GetGroupsEnabledModels([]string{name}), item.ModelName) {
					choice.Groups[name] = service.GetUserGroupRatio(group, name)
				}
			}
		}
		if len(choice.Groups) > 0 {
			result = append(result, choice)
		}
	}
	return result
}

func imageStudioPublicConfig(c *gin.Context) (model.ImageStudioConfig, bool) {
	config, err := model.GetImageStudioConfig()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return config, false
	}
	return config, true
}

func GetImageStudioOptions(c *gin.Context) {
	config, ok := imageStudioPublicConfig(c)
	if !ok {
		return
	}
	models := imageStudioCatalog(config, c.GetInt("id"), false)
	c.JSON(200, gin.H{
		"models": models, "ready": config.OperatorName != "" && config.ContactEmail != "" && len(models) != 0,
		"operator_name": config.OperatorName, "contact_email": config.ContactEmail,
		"agreement_revision": model.ImageStudioRevision(config.Agreement),
		"privacy_revision":   model.ImageStudioRevision(config.Privacy),
		"retention_days":     config.RetentionDays,
	})
}

func GetImageStudioLegal(c *gin.Context) {
	config, ok := imageStudioPublicConfig(c)
	if !ok {
		return
	}
	content := config.Agreement
	if c.Param("kind") == "privacy" {
		content = config.Privacy
	} else if c.Param("kind") != "agreement" {
		c.Status(404)
		return
	}
	c.JSON(200, gin.H{
		"content": content, "revision": model.ImageStudioRevision(content),
		"operator_name": config.OperatorName, "contact_email": config.ContactEmail,
	})
}

func GetImageStudioAdminConfig(c *gin.Context) {
	config, ok := imageStudioPublicConfig(c)
	if !ok {
		return
	}
	c.JSON(200, gin.H{"config": config, "models": config.EnabledModels(), "candidates": imageStudioCatalog(config, c.GetInt("id"), true)})
}

func UpdateImageStudioAdminConfig(c *gin.Context) {
	var request struct {
		Models        []model.ImageStudioModel `json:"models"`
		OperatorName  string                   `json:"operator_name"`
		ContactEmail  string                   `json:"contact_email"`
		Agreement     string                   `json:"agreement"`
		Privacy       string                   `json:"privacy"`
		StorageMode   string                   `json:"storage_mode"`
		LocalDir      string                   `json:"local_dir"`
		S3Endpoint    string                   `json:"s3_endpoint"`
		S3Bucket      string                   `json:"s3_bucket"`
		S3Region      string                   `json:"s3_region"`
		S3AccessKey   string                   `json:"s3_access_key"`
		S3SecretKey   string                   `json:"s3_secret_key"`
		RetentionDays int                      `json:"retention_days"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(request.Agreement) == "" || strings.TrimSpace(request.Privacy) == "" ||
		len(request.Agreement) > 20000 || len(request.Privacy) > 20000 ||
		len(request.OperatorName) > 200 || len(request.ContactEmail) > 254 {
		c.JSON(400, gin.H{"error": "invalid legal documents or operator details"})
		return
	}
	if request.ContactEmail != "" {
		if _, err := mail.ParseAddress(request.ContactEmail); err != nil {
			c.JSON(400, gin.H{"error": "invalid contact email"})
			return
		}
	}
	config, ok := imageStudioPublicConfig(c)
	if !ok {
		return
	}
	candidates := imageStudioCatalog(config, c.GetInt("id"), true)
	seen := map[string]bool{}
	for _, selected := range request.Models {
		if seen[selected.Name] || !slices.ContainsFunc(candidates, func(v imageStudioModelInfo) bool { return v.Name == selected.Name }) {
			c.JSON(400, gin.H{"error": "model is not an eligible fixed-price image model"})
			return
		}
		seen[selected.Name] = true
	}
	modelJSON, _ := common.Marshal(request.Models)
	config.Models = string(modelJSON)
	config.Agreement, config.Privacy = request.Agreement, request.Privacy
	config.OperatorName, config.ContactEmail = strings.TrimSpace(request.OperatorName), strings.TrimSpace(request.ContactEmail)
	if c.GetInt("role") >= common.RoleRootUser {
		if request.RetentionDays < 1 || request.RetentionDays > 180 ||
			(request.StorageMode != "local" && request.StorageMode != "s3") {
			c.JSON(400, gin.H{"error": "invalid storage configuration"})
			return
		}
		if request.StorageMode == "local" && strings.TrimSpace(request.LocalDir) == "" {
			c.JSON(400, gin.H{"error": "storage directory is required"})
			return
		}
		if request.StorageMode == "s3" {
			if request.S3Endpoint == "" || request.S3Bucket == "" || request.S3Region == "" ||
				(request.S3AccessKey == "" && config.S3AccessKey == "") ||
				(request.S3SecretKey == "" && config.S3SecretKey == "") {
				c.JSON(400, gin.H{"error": "S3 endpoint, bucket, region and credentials are required"})
				return
			}
		}
		config.StorageMode, config.LocalDir, config.RetentionDays = request.StorageMode, request.LocalDir, request.RetentionDays
		config.S3Endpoint, config.S3Bucket, config.S3Region = request.S3Endpoint, request.S3Bucket, request.S3Region
		if request.S3AccessKey != "" {
			config.S3AccessKey = request.S3AccessKey
		}
		if request.S3SecretKey != "" {
			config.S3SecretKey = request.S3SecretKey
		}
		if config.StorageMode == "s3" {
			if err := system_setting.ValidateTaskArtifactStoreConfig(system_setting.TaskArtifactStoreConfig{
				Mode:                system_setting.TaskArtifactStoreModeS3,
				S3Endpoint:          config.S3Endpoint,
				S3Bucket:            config.S3Bucket,
				S3Region:            config.S3Region,
				S3AccessKey:         config.S3AccessKey,
				S3SecretKey:         config.S3SecretKey,
				S3PresignTTLSeconds: system_setting.DefaultTaskArtifactStorePresignTTLSeconds,
			}); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}
	if err := model.DB.Save(&config).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

type imageStudioRequest struct {
	Model             string `form:"model"`
	Group             string `form:"group"`
	Prompt            string `form:"prompt"`
	Size              string `form:"size"`
	N                 string `form:"n"`
	AgreementRevision string `form:"agreement_revision"`
	PrivacyRevision   string `form:"privacy_revision"`
	Consent           string `form:"consent"`
}

type imageStudioCapture struct {
	gin.ResponseWriter
	body     bytes.Buffer
	status   int
	header   http.Header
	exceeded bool
}

func (w *imageStudioCapture) Header() http.Header  { return w.header }
func (w *imageStudioCapture) WriteHeader(code int) { w.status = code }
func (w *imageStudioCapture) WriteHeaderNow()      {}
func (w *imageStudioCapture) Write(data []byte) (int, error) {
	if w.body.Len()+len(data) > 100<<20 {
		w.exceeded = true
		return 0, errors.New("image studio response exceeds 100 MB")
	}
	return w.body.Write(data)
}
func (w *imageStudioCapture) WriteString(data string) (int, error) { return w.Write([]byte(data)) }
func (w *imageStudioCapture) Status() int                          { return w.status }
func (w *imageStudioCapture) Flush()                               {}

var imageStudioSizePattern = regexp.MustCompile(`^([1-9][0-9]*)x([1-9][0-9]*)$`)

func ImageStudioPrepare(c *gin.Context) {
	config, ok := imageStudioPublicConfig(c)
	if !ok {
		c.Abort()
		return
	}
	if config.OperatorName == "" || config.ContactEmail == "" {
		c.AbortWithStatusJSON(403, gin.H{"error": "image studio requires operator details"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 42<<20)
	if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid or oversized request"})
		return
	}
	incomingForm := c.Request.MultipartForm
	defer incomingForm.RemoveAll()
	var req imageStudioRequest
	if err := c.ShouldBind(&req); err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid request"})
		return
	}
	if req.Consent != "true" || req.AgreementRevision != model.ImageStudioRevision(config.Agreement) ||
		req.PrivacyRevision != model.ImageStudioRevision(config.Privacy) {
		c.AbortWithStatusJSON(409, gin.H{"error": "please read and accept the current image studio terms"})
		return
	}
	n := 1
	if req.N != "" {
		var err error
		n, err = strconv.Atoi(req.N)
		if err != nil || n < 1 || n > dto.MaxImageN {
			c.AbortWithStatusJSON(400, gin.H{"error": "invalid image count"})
			return
		}
	}
	if req.Size != "" {
		matches := imageStudioSizePattern.FindStringSubmatch(req.Size)
		if matches == nil {
			c.AbortWithStatusJSON(400, gin.H{"error": "invalid image size"})
			return
		}
		width, _ := strconv.Atoi(matches[1])
		height, _ := strconv.Atoi(matches[2])
		if width < 256 || width > 4096 || height < 256 || height > 4096 {
			c.AbortWithStatusJSON(400, gin.H{"error": "image dimensions must be between 256 and 4096"})
			return
		}
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" || len(req.Prompt) > 10000 {
		c.AbortWithStatusJSON(400, gin.H{"error": "prompt is required (up to 10000 bytes)"})
		return
	}
	var selected *imageStudioModelInfo
	for _, candidate := range imageStudioCatalog(config, c.GetInt("id"), false) {
		if candidate.Name == req.Model {
			copy := candidate
			selected = &copy
			break
		}
	}
	if selected == nil {
		c.AbortWithStatusJSON(403, gin.H{"error": "model is unavailable"})
		return
	}
	ratio, available := selected.Groups[req.Group]
	if !available || ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		c.AbortWithStatusJSON(403, gin.H{"error": "group is unavailable for this model"})
		return
	}
	estimate := selected.Price * ratio * float64(n)
	if math.IsNaN(estimate) || math.IsInf(estimate, 0) {
		c.AbortWithStatusJSON(400, gin.H{"error": "image price is unavailable"})
		return
	}
	references := c.Request.MultipartForm.File["image"]
	if len(references) > 4 || len(references) > 0 && !selected.AllowEdits {
		c.AbortWithStatusJSON(400, gin.H{"error": "this model does not accept these reference images"})
		return
	}
	originals := make([][]byte, 0, len(references))
	originalMimes := make([]string, 0, len(references))
	thumbnails := make([][]byte, 0, len(references))
	for _, file := range references {
		if file.Size < 1 || file.Size > 10<<20 {
			c.AbortWithStatusJSON(400, gin.H{"error": "reference image must be at most 10 MB"})
			return
		}
		stream, err := file.Open()
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": "invalid reference image"})
			return
		}
		data, err := io.ReadAll(io.LimitReader(stream, 10<<20+1))
		stream.Close()
		if err != nil || len(data) > 10<<20 {
			c.AbortWithStatusJSON(400, gin.H{"error": "invalid reference image"})
			return
		}
		mime := http.DetectContentType(data)
		if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
			c.AbortWithStatusJSON(400, gin.H{"error": "reference must be PNG, JPEG or WebP"})
			return
		}
		dimensions, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || dimensions.Width < 1 || dimensions.Height < 1 || int64(dimensions.Width)*int64(dimensions.Height) > 32_000_000 {
			c.AbortWithStatusJSON(400, gin.H{"error": "reference dimensions are invalid"})
			return
		}
		source, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": "reference cannot be decoded"})
			return
		}
		scale := math.Min(256/float64(dimensions.Width), 256/float64(dimensions.Height))
		scale = math.Min(scale, 1)
		thumb := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(dimensions.Width)*scale)), max(1, int(float64(dimensions.Height)*scale))))
		draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), source, source.Bounds(), draw.Over, nil)
		var output bytes.Buffer
		if err := jpeg.Encode(&output, thumb, &jpeg.Options{Quality: 76}); err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": "thumbnail encoding failed"})
			return
		}
		originals, thumbnails = append(originals, data), append(thumbnails, output.Bytes())
		originalMimes = append(originalMimes, mime)
	}
	record := model.ImageStudioRecord{
		UserID: c.GetInt("id"), Model: req.Model, Group: req.Group, Prompt: req.Prompt,
		Size: req.Size, Count: n, EstimatedPrice: estimate,
		Status: "processing", AgreementRevision: req.AgreementRevision, PrivacyRevision: req.PrivacyRevision,
		AcceptedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Duration(config.RetentionDays) * 24 * time.Hour),
	}
	if err := model.DB.Create(&record).Error; err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": "could not create generation record"})
		return
	}
	for _, thumbnail := range thumbnails {
		if _, err := service.StoreImageStudioAsset(c.Request.Context(), config, record.ID, "reference", "image/jpeg", thumbnail); err != nil {
			_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "could not save reference thumbnail"}).Error
			c.AbortWithStatusJSON(500, gin.H{"error": "could not save reference thumbnail"})
			return
		}
	}
	var body bytes.Buffer
	path := "/v1/images/generations"
	if len(originals) > 0 {
		path = "/v1/images/edits"
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("model", req.Model)
		_ = writer.WriteField("prompt", req.Prompt)
		_ = writer.WriteField("n", strconv.Itoa(n))
		if req.Size != "" {
			_ = writer.WriteField("size", req.Size)
		}
		for i, data := range originals {
			extension := ".png"
			if originalMimes[i] == "image/jpeg" {
				extension = ".jpg"
			} else if originalMimes[i] == "image/webp" {
				extension = ".webp"
			}
			part, err := writer.CreateFormFile("image", fmt.Sprintf("reference-%d%s", i, extension))
			if err != nil {
				_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "could not prepare reference image"}).Error
				c.AbortWithStatus(500)
				return
			}
			if _, err := part.Write(data); err != nil {
				_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "could not prepare reference image"}).Error
				c.AbortWithStatus(500)
				return
			}
		}
		_ = writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	} else {
		requestJSON := map[string]any{"model": req.Model, "prompt": req.Prompt, "n": n, "response_format": "b64_json"}
		if req.Size != "" {
			requestJSON["size"] = req.Size
		}
		data, _ := common.Marshal(requestJSON)
		body.Write(data)
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Request.URL.Path = path
	common.CleanupBodyStorage(c)
	c.Request.Body = io.NopCloser(&body)
	c.Request.ContentLength = int64(body.Len())
	c.Request.MultipartForm, c.Request.PostForm, c.Request.Form = nil, nil, nil
	userCache, err := model.GetUserCache(c.GetInt("id"))
	if err != nil {
		_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "user unavailable"}).Error
		c.AbortWithStatusJSON(500, gin.H{"error": "user unavailable"})
		return
	}
	userCache.WriteContext(c)
	token := &model.Token{UserId: record.UserID, Name: "image-studio", Group: req.Group}
	if err := middleware.SetupContextForToken(c, token); err != nil {
		_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "could not initialize billing"}).Error
		c.AbortWithStatus(500)
		return
	}
	common.SetContextKey(c, constant.ContextKeyUsingGroup, req.Group)
	originalWriter := c.Writer
	capture := &imageStudioCapture{ResponseWriter: originalWriter, status: 200, header: originalWriter.Header().Clone()}
	c.Writer = capture
	c.Set("image_studio", true)
	c.Next()
	c.Writer = originalWriter
	if quota, exists := c.Get("image_studio_billed_quota"); exists {
		if billed, ok := quota.(int); ok {
			actual := float64(billed) / common.QuotaPerUnit
			_ = model.DB.Model(&record).Update("actual_price", actual).Error
		}
	}
	if capture.exceeded {
		_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "generated response exceeded 100 MB"}).Error
		c.JSON(http.StatusBadGateway, gin.H{"error": "generated response exceeded 100 MB"})
		return
	}
	if capture.status >= 400 || capture.body.Len() == 0 {
		message := "image generation failed"
		if capture.body.Len() > 0 {
			message = string(capture.body.Bytes())
			if len(message) > 1000 {
				message = message[:1000]
			}
		}
		message = strings.ReplaceAll(message, req.Prompt, "[prompt]")
		_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": message}).Error
		if capture.body.Len() == 0 {
			c.JSON(502, gin.H{"error": message})
			return
		}
		c.Data(max(capture.status, 400), "application/json", capture.body.Bytes())
		return
	}
	var response dto.ImageResponse
	if err := common.Unmarshal(capture.body.Bytes(), &response); err != nil || len(response.Data) == 0 {
		_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "upstream returned no downloadable images"}).Error
		c.JSON(502, gin.H{"error": "upstream returned no downloadable images"})
		return
	}
	for _, item := range response.Data {
		var imageBytes []byte
		var err error
		if item.B64Json != "" {
			imageBytes, err = base64.StdEncoding.DecodeString(item.B64Json)
		} else if item.Url != "" {
			imageBytes, err = service.FetchImageStudioResult(c.Request.Context(), item.Url)
		}
		if err != nil || len(imageBytes) > 50<<20 {
			_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "result download failed"}).Error
			c.JSON(502, gin.H{"error": "result download failed"})
			return
		}
		mime := http.DetectContentType(imageBytes)
		if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
			_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "result was not an image"}).Error
			c.JSON(502, gin.H{"error": "result was not an image"})
			return
		}
		if _, err := service.StoreImageStudioAsset(c.Request.Context(), config, record.ID, "result", mime, imageBytes); err != nil {
			_ = model.DB.Model(&record).Updates(map[string]any{"status": "failed", "error": "could not store generated result"}).Error
			c.JSON(500, gin.H{"error": "could not store generated result"})
			return
		}
	}
	_ = model.DB.Model(&record).Update("status", "completed").Error
	c.JSON(200, gin.H{"id": record.ID, "status": "completed"})
}

func ImageStudioRelay(c *gin.Context) { Relay(c, types.RelayFormatOpenAIImage) }

func ListImageStudioRecords(c *gin.Context) {
	query := model.DB.Model(&model.ImageStudioRecord{})
	if c.FullPath() != "/api/image-studio/admin/records" {
		query = query.Where("user_id = ?", c.GetInt("id"))
	} else if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	page = min(page, 100000)
	var total int64
	query.Count(&total)
	var records []model.ImageStudioRecord
	if err := query.Order("id desc").Limit(20).Offset((page - 1) * 20).Find(&records).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	type entry struct {
		model.ImageStudioRecord
		Assets []model.ImageStudioAsset `json:"assets"`
	}
	result := make([]entry, 0, len(records))
	for _, record := range records {
		var assets []model.ImageStudioAsset
		if time.Now().Before(record.ExpiresAt) {
			_ = model.DB.Where("record_id = ?", record.ID).Find(&assets).Error
		}
		result = append(result, entry{record, assets})
	}
	c.JSON(200, gin.H{"data": result, "total": total, "page": page})
}

func imageStudioOwnedRecord(c *gin.Context) (*model.ImageStudioRecord, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return nil, false
	}
	var record model.ImageStudioRecord
	if err := model.DB.Where("id = ?", id).First(&record).Error; err != nil ||
		(c.GetInt("role") < common.RoleAdminUser || !strings.Contains(c.FullPath(), "/admin/")) && record.UserID != c.GetInt("id") {
		c.JSON(404, gin.H{"error": "record not found"})
		return nil, false
	}
	return &record, true
}

func GetImageStudioAsset(c *gin.Context) {
	record, ok := imageStudioOwnedRecord(c)
	if !ok {
		return
	}
	if time.Now().After(record.ExpiresAt) {
		c.JSON(410, gin.H{"error": "image expired"})
		return
	}
	var asset model.ImageStudioAsset
	if err := model.DB.Where("id = ? AND record_id = ?", c.Param("asset"), record.ID).First(&asset).Error; err != nil {
		c.Status(404)
		return
	}
	data, err := service.ReadImageStudioAsset(c.Request.Context(), &asset)
	if err != nil || len(data) > 50<<20 {
		c.JSON(500, gin.H{"error": "image unavailable"})
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	if c.Query("download") == "1" {
		extension := ".png"
		switch asset.MimeType {
		case "image/jpeg":
			extension = ".jpg"
		case "image/webp":
			extension = ".webp"
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"image-%d-%d%s\"", record.ID, asset.ID, extension))
	}
	c.Data(200, asset.MimeType, data)
}

func DeleteImageStudioRecord(c *gin.Context) {
	record, ok := imageStudioOwnedRecord(c)
	if !ok {
		return
	}
	if record.Status == "processing" {
		c.JSON(http.StatusConflict, gin.H{"error": "generation is still processing"})
		return
	}
	if err := service.PurgeImageStudioRecord(c.Request.Context(), record, true); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func BatchImageStudioRecords(c *gin.Context) {
	var input struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.IDs) == 0 || len(input.IDs) > 100 {
		c.JSON(400, gin.H{"error": "select 1-100 records"})
		return
	}
	var records []model.ImageStudioRecord
	if err := model.DB.Where("id IN ?", input.IDs).Find(&records).Error; err != nil || len(records) != len(input.IDs) {
		c.JSON(400, gin.H{"error": "invalid record selection"})
		return
	}
	if c.Param("action") == "delete" {
		for _, record := range records {
			if record.Status == "processing" {
				c.JSON(http.StatusConflict, gin.H{"error": "generation is still processing"})
				return
			}
		}
		for i := range records {
			if err := service.PurgeImageStudioRecord(c.Request.Context(), &records[i], true); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(200, gin.H{"success": true})
		return
	}
	if c.Param("action") != "zip" {
		c.Status(404)
		return
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	totalBytes := 0
	for _, record := range records {
		if time.Now().After(record.ExpiresAt) {
			continue
		}
		var assets []model.ImageStudioAsset
		_ = model.DB.Where("record_id = ? AND kind = ?", record.ID, "result").Find(&assets).Error
		for _, asset := range assets {
			data, err := service.ReadImageStudioAsset(c.Request.Context(), &asset)
			totalBytes += len(data)
			if err != nil || totalBytes > 100<<20 {
				writer.Close()
				c.JSON(413, gin.H{"error": "archive exceeds 100 MB or contains an unavailable image"})
				return
			}
			name := fmt.Sprintf("image-%d-%d", record.ID, asset.ID)
			switch asset.MimeType {
			case "image/png":
				name += ".png"
			case "image/jpeg":
				name += ".jpg"
			case "image/webp":
				name += ".webp"
			}
			file, err := writer.Create(name)
			if err != nil {
				c.Status(500)
				return
			}
			if _, err := file.Write(data); err != nil {
				c.Status(500)
				return
			}
		}
	}
	if err := writer.Close(); err != nil {
		c.Status(500)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\"image-studio.zip\"")
	c.Data(200, "application/zip", archive.Bytes())
}
