package controller

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAllModelsMeta 获取模型列表（分页）
func GetAllModelsMeta(c *gin.Context) {
	listModelsMeta(c, "", "")
}

// SearchModelsMeta 搜索模型列表
func SearchModelsMeta(c *gin.Context) {
	listModelsMeta(c, c.Query("keyword"), c.Query("vendor"))
}

func listModelsMeta(c *gin.Context, keyword, vendor string) {
	squareState := model.ModelSquareState(c.Query("square_state"))
	switch squareState {
	case "", model.ModelSquareVisible, model.ModelSquareUnavailable, model.ModelSquareHidden, model.ModelSquarePartial:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid model square state"})
		return
	}

	pageInfo := common.GetPageQuery(c)
	if squareState != "" && (pageInfo.GetPage() < 1 || pageInfo.GetPageSize() < 1) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid pagination"})
		return
	}
	offset, limit := pageInfo.GetStartIdx(), pageInfo.GetPageSize()
	if squareState != "" {
		// Visibility depends on live channels and metadata rules. Filter the
		// enriched candidate set before counting and paginating the results.
		offset, limit = 0, -1
	}
	search := model.SearchModels
	if c.Query("include_channel_models") == "true" {
		search = model.SearchModelsWithChannels
	}
	modelsMeta, total, err := search(keyword, vendor, c.Query("status"), c.Query("sync_official"), offset, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := enrichModels(modelsMeta); err != nil {
		common.ApiError(c, err)
		return
	}
	if squareState != "" {
		filtered := make([]*model.Model, 0, len(modelsMeta))
		for _, metadata := range modelsMeta {
			if metadata.SquareState == squareState {
				filtered = append(filtered, metadata)
			}
		}
		total = int64(len(filtered))
		start := len(filtered)
		if pageInfo.GetPage()-1 <= len(filtered)/pageInfo.GetPageSize() {
			start = (pageInfo.GetPage() - 1) * pageInfo.GetPageSize()
		}
		end := min(start+pageInfo.GetPageSize(), len(filtered))
		modelsMeta = filtered[start:end]
	}

	vendorCounts, _ := model.GetVendorModelCounts()
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(modelsMeta)
	common.ApiSuccess(c, gin.H{
		"items":         modelsMeta,
		"total":         total,
		"page":          pageInfo.GetPage(),
		"page_size":     pageInfo.GetPageSize(),
		"vendor_counts": vendorCounts,
	})
}

// GetModelMeta 根据 ID 获取单条模型信息
func GetModelMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var m model.Model
	if err := model.DB.First(&m, id).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if err := enrichModels([]*model.Model{&m}); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &m)
}

// CreateModelMeta 新建模型
func CreateModelMeta(c *gin.Context) {
	var m model.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		common.ApiError(c, err)
		return
	}
	if m.ModelName == "" {
		common.ApiErrorMsg(c, "模型名称不能为空")
		return
	}
	if err := model.ValidateMetadataValues(model.MetadataValues{Endpoints: m.Endpoints, Status: m.Status, NameRule: m.NameRule}); err != nil {
		common.ApiError(c, err)
		return
	}
	// 名称冲突检查
	if dup, err := model.IsModelNameDuplicated(0, m.ModelName); err != nil {
		common.ApiError(c, err)
		return
	} else if dup {
		common.ApiErrorMsg(c, "模型名称已存在")
		return
	}

	if err := m.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	res := service.SyncModelChannelAvailabilityAfterMutation("model.create")
	if !res.PricingRefreshed {
		model.RefreshPricing()
	}
	if err := reloadModelMetaAfterMutation(&m); err != nil {
		common.ApiError(c, err)
		return
	}
	m.HasMetadata = m.Id > 0
	common.ApiSuccess(c, &m)
}

func reloadModelMetaAfterMutation(current *model.Model) error {
	if current == nil || current.Id == 0 {
		return nil
	}
	var persisted model.Model
	if err := model.DB.First(&persisted, current.Id).Error; err != nil {
		common.SysError(fmt.Sprintf("failed to reload model metadata after mutation: id=%d err=%v", current.Id, err))
		return fmt.Errorf("reload model metadata after mutation: %w", err)
	}
	*current = persisted
	return nil
}

// UpdateModelMeta 更新模型
func UpdateModelMeta(c *gin.Context) {
	statusOnly := c.Query("status_only") == "true"

	var m model.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		common.ApiError(c, err)
		return
	}
	if m.Id == 0 {
		common.ApiErrorMsg(c, "缺少模型 ID")
		return
	}

	if statusOnly {
		if m.Status != 0 && m.Status != 1 {
			common.ApiErrorMsg(c, "invalid catalog visibility")
			return
		}
		// 只更新状态，防止误清空其他字段
		updateResult := model.DB.Model(&model.Model{}).Where("id = ?", m.Id).Updates(map[string]any{
			"status":                m.Status,
			"auto_disabled_by_rule": false,
			"updated_time":          common.GetTimestamp(),
		})
		if updateResult.Error != nil {
			common.ApiError(c, updateResult.Error)
			return
		}
		if updateResult.RowsAffected == 0 {
			var count int64
			if err := model.DB.Model(&model.Model{}).Where("id = ?", m.Id).Count(&count).Error; err != nil {
				common.ApiError(c, err)
				return
			}
			if count == 0 {
				common.ApiError(c, gorm.ErrRecordNotFound)
				return
			}
		}
		// Re-evaluate immediately so auto-disable can correct a manual enable without channels.
		res := service.SyncModelChannelAvailabilityAfterMutation("model.status_update")
		if !res.PricingRefreshed {
			model.RefreshPricing()
		}
		if err := reloadModelMetaAfterMutation(&m); err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, &m)
		return
	}

	if strings.TrimSpace(m.ModelName) == "" {
		common.ApiErrorMsg(c, "模型名称不能为空")
		return
	}
	if err := model.ValidateMetadataValues(model.MetadataValues{Endpoints: m.Endpoints, Status: m.Status, NameRule: m.NameRule}); err != nil {
		common.ApiError(c, err)
		return
	}
	// 名称冲突检查
	if dup, err := model.IsModelNameDuplicated(m.Id, m.ModelName); err != nil {
		common.ApiError(c, err)
		return
	} else if dup {
		common.ApiErrorMsg(c, "模型名称已存在")
		return
	}

	if err := m.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	res := service.SyncModelChannelAvailabilityAfterMutation("model.update")
	if !res.PricingRefreshed {
		model.RefreshPricing()
	}
	if err := reloadModelMetaAfterMutation(&m); err != nil {
		common.ApiError(c, err)
		return
	}
	m.HasMetadata = m.Id > 0
	common.ApiSuccess(c, &m)
}

type batchUpdateModelStatusRequest struct {
	Ids    []int `json:"ids"`
	Status int   `json:"status"`
}

// BatchUpdateModelStatus updates selected models and reconciles channel
// availability once for the whole batch.
func BatchUpdateModelStatus(c *gin.Context) {
	req := batchUpdateModelStatusRequest{}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 || (req.Status != 0 && req.Status != 1) {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if len(req.Ids) > 100 {
		common.ApiErrorI18n(c, i18n.MsgBatchTooMany, map[string]any{"Max": 100})
		return
	}

	ids := make([]int, 0, len(req.Ids))
	seen := make(map[int]struct{}, len(req.Ids))
	for _, id := range req.Ids {
		if id <= 0 {
			common.ApiErrorI18n(c, i18n.MsgInvalidParams)
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if err := model.DB.Model(&model.Model{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":                req.Status,
			"auto_disabled_by_rule": false,
			"updated_time":          common.GetTimestamp(),
		}).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	res := service.SyncModelChannelAvailabilityAfterMutation("model.status_update_batch")
	if !res.PricingRefreshed {
		model.RefreshPricing()
	}

	var persisted []model.Model
	if err := model.DB.Select("id", "status").Where("id IN ?", ids).Find(&persisted).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	statusByID := make(map[int]int, len(persisted))
	for i := range persisted {
		statusByID[persisted[i].Id] = persisted[i].Status
	}
	updated := 0
	failedIds := make([]int, 0)
	for _, id := range ids {
		if status, ok := statusByID[id]; ok && status == req.Status {
			updated++
			continue
		}
		failedIds = append(failedIds, id)
	}

	common.ApiSuccess(c, gin.H{
		"updated":    updated,
		"failed_ids": failedIds,
	})
}

// DeleteModelMeta 删除模型
func DeleteModelMeta(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	removeFromChannels, err := strconv.ParseBool(c.DefaultQuery("remove_from_channels", "false"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	removePricing, err := strconv.ParseBool(c.DefaultQuery("remove_pricing", "false"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if removePricing && c.GetInt("role") != common.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Model pricing is managed by a super administrator."})
		return
	}
	result, err := model.DeleteModelMetadata([]int{id}, removeFromChannels, removePricing)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "model.delete", map[string]any{"model_ids": []int{id}, "remove_from_channels": removeFromChannels, "remove_pricing": removePricing, "updated_channels": result.UpdatedChannels})
	res := service.SyncModelChannelAvailabilityAfterMutation("model.delete")
	if !res.PricingRefreshed {
		model.RefreshPricing()
	}
	common.ApiSuccess(c, result)
}

func BatchDeleteModelMeta(c *gin.Context) {
	var request struct {
		ModelIDs           []int `json:"model_ids"`
		RemoveFromChannels bool  `json:"remove_from_channels"`
		RemovePricing      bool  `json:"remove_pricing"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}
	if request.RemovePricing && c.GetInt("role") != common.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Model pricing is managed by a super administrator."})
		return
	}
	result, err := model.DeleteModelMetadata(request.ModelIDs, request.RemoveFromChannels, request.RemovePricing)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "model.delete_batch", map[string]any{"model_ids": request.ModelIDs, "remove_from_channels": request.RemoveFromChannels, "remove_pricing": request.RemovePricing, "updated_channels": result.UpdatedChannels})
	res := service.SyncModelChannelAvailabilityAfterMutation("model.delete_batch")
	if !res.PricingRefreshed {
		model.RefreshPricing()
	}
	common.ApiSuccess(c, result)
}

// enrichModels keeps configured endpoints intact and derives connections from
// enabled routes, including hidden or unpriced models absent from the catalog.
func enrichModels(models []*model.Model) error {
	if len(models) == 0 {
		return nil
	}
	configured, err := model.GetConfiguredModelChannels()
	if err != nil {
		return err
	}
	for _, metadata := range models {
		if metadata == nil {
			continue
		}
		metadata.HasMetadata = metadata.Id > 0
		channelIDs := make(map[int]struct{})
		for name, ids := range configured {
			if metadata.MatchesName(name) {
				for _, id := range ids {
					channelIDs[id] = struct{}{}
				}
			}
		}
		metadata.ConfiguredChannelCount = len(channelIDs)
	}
	connections, err := model.GetModelConnections()
	if err != nil {
		return err
	}
	if err := model.FillModelSquareStates(models, configured, connections); err != nil {
		return err
	}
	for _, metadata := range models {
		if metadata == nil {
			continue
		}
		channels := make(map[int]model.BoundChannel)
		groups := make(map[string]bool)
		names := make(map[string]bool)
		endpoints := make(map[string]bool)
		quotas := make(map[int]bool)
		for _, connection := range connections {
			name := connection.Model
			if !metadata.MatchesName(name) {
				continue
			}
			names[name] = true
			groups[connection.Group] = true
			channels[connection.ChannelId] = model.BoundChannel{Name: connection.ChannelName, Type: connection.ChannelType}
			for _, endpoint := range model.GetModelSupportEndpointTypes(name) {
				endpoints[string(endpoint)] = true
			}
			for _, quota := range model.GetModelQuotaTypes(name) {
				quotas[quota] = true
			}
		}
		metadata.BoundChannels = nil
		metadata.EnableGroups = nil
		metadata.SupportedEndpoints = nil
		metadata.QuotaTypes = nil
		metadata.MatchedModels = nil
		for _, channel := range channels {
			metadata.BoundChannels = append(metadata.BoundChannels, channel)
		}
		sort.Slice(metadata.BoundChannels, func(i, j int) bool {
			a, b := metadata.BoundChannels[i], metadata.BoundChannels[j]
			if a.Name == b.Name {
				return a.Type < b.Type
			}
			return a.Name < b.Name
		})
		for group := range groups {
			metadata.EnableGroups = append(metadata.EnableGroups, group)
		}
		for endpoint := range endpoints {
			metadata.SupportedEndpoints = append(metadata.SupportedEndpoints, endpoint)
		}
		for quota := range quotas {
			metadata.QuotaTypes = append(metadata.QuotaTypes, quota)
		}
		sort.Strings(metadata.EnableGroups)
		sort.Strings(metadata.SupportedEndpoints)
		sort.Ints(metadata.QuotaTypes)
		if metadata.NameRule != model.NameRuleExact {
			for name := range names {
				metadata.MatchedModels = append(metadata.MatchedModels, name)
			}
			sort.Strings(metadata.MatchedModels)
			metadata.MatchedCount = len(names)
		}
	}
	return nil
}

// BatchDisableModelsNoChannels 批量禁用无可用渠道的模型
func BatchDisableModelsNoChannels(c *gin.Context) {
	result, err := service.ManualDisableModelsWithoutChannels()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"disabled": result.Disabled,
		"enabled":  result.Enabled,
		"skipped":  result.Skipped,
		"reason":   result.Reason,
	})
}

// BatchEnableModelsWithChannels 批量启用：仅恢复被渠道可用性规则自动禁用、且现已有可用渠道的模型
func BatchEnableModelsWithChannels(c *gin.Context) {
	result, err := service.ManualEnableModelsWithChannels()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"disabled": result.Disabled,
		"enabled":  result.Enabled,
		"skipped":  result.Skipped,
		"reason":   result.Reason,
	})
}
