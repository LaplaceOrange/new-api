package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const DefaultImageStudioAgreement = `1. **适用范围。** 本协议仅适用于本站控制台的在线生图功能。每次提交生成请求前，请阅读本协议及《在线生图隐私政策》，并主动勾选同意。
2. **输入内容。** 您应确保对提交的提示词和参考图享有必要权利或授权，不得利用本功能侵害他人的隐私、肖像、知识产权等合法权益，或提交违反适用法律及所选模型服务规则的内容。
3. **生成服务。** 本功能将您的输入发送至所选模型对应的服务提供者进行处理。不同模型支持的尺寸、参考图及生成张数可能不同；请求可能因模型能力、内容审核或服务状态而失败。生成结果不保证准确、独创、排他或适合特定用途。
4. **费用。** 提交前显示按当前模型价格、用户分组倍率和张数计算的预估费用；提交时系统再次核算。实际扣费及失败请求的额度处理以本站账单和现有结算规则为准。模型、价格和倍率可能依配置调整，调整不追溯已完成的请求。
5. **作品与期限。** 您可在作品有效期内查看和下载生成图片；到期或主动删除后，本站不保证能够恢复。您使用、发布或商业利用生成内容时，应自行确认适用法律、第三方权益及模型提供者的使用条件。
6. **变更与联系。** 运营主体可根据功能和处理方式更新本协议；更新后须重新阅读并同意才能继续生成。有关本功能的疑问，可通过本页公示的联系邮箱提出。`

const DefaultImageStudioPrivacy = `1. **适用范围。** 本政策仅说明在线生图功能如何处理您的信息，不替代或修改本站其他服务的隐私政策。
2. **处理的信息与目的。** 为完成生成、展示记录、计费和保障服务正常运行，我们处理您的账号标识、提示词、可选参考图、生成图片、所选模型与分组、尺寸、张数、费用、提交时间、任务状态及必要的错误记录。不提供提示词及必要参数将无法发起生成；不上传参考图仍可使用支持文生图的模型。
3. **参考图与生成结果。** 参考图原件用于本次模型请求，不保存为历史作品；为便于识别记录，本站仅保存其缩略图。生成图片存放于受访问控制的存储空间，只有您本人和获授权的管理员可通过本功能查看；管理员可按管理权限处理记录。
4. **接收方与传输。** 提示词、必要参数及您选择上传的参考图会传送给所选模型对应的上游服务提供者；图片可能保存在运营主体配置的存储服务中。服务提供者可能位于其他地区，并可能依其自身规则处理收到的信息。实际供应商及处理地区由运营主体配置，您可通过本页联系邮箱询问；请勿提交无权提供或不愿传送给上游的敏感内容。
5. **保存与删除。** 图片和参考图缩略图默认保存 30 天，实际到期时间以生成记录所示为准；到期时一并删除对应提示词。不含提示词的记录保留至生成后 180 天，以便查询任务和处理争议，随后清理。您可提前删除自己的作品与记录；既有账单记录及上游服务提供者已接收的信息，不因删除本功能记录而同步消除。
6. **您的选择与联系。** 您可查看、下载和删除有效期内的作品；也可通过页首联系邮箱提出与本功能相关的访问、更正或删除请求。若本政策更新，在您再次使用生图功能前将展示新版本并要求重新同意。`

type ImageStudioModel struct {
	Name       string `json:"name"`
	AllowEdits bool   `json:"allow_edits"`
}

type ImageStudioConfig struct {
	ID            int    `gorm:"primaryKey" json:"-"`
	Models        string `gorm:"type:text" json:"-"`
	OperatorName  string `gorm:"type:text" json:"operator_name"`
	ContactEmail  string `gorm:"type:text" json:"contact_email"`
	Agreement     string `gorm:"type:text" json:"agreement"`
	Privacy       string `gorm:"type:text" json:"privacy"`
	StorageMode   string `json:"storage_mode"`
	LocalDir      string `gorm:"type:text" json:"local_dir"`
	S3Endpoint    string `gorm:"type:text" json:"s3_endpoint"`
	S3Bucket      string `gorm:"type:text" json:"s3_bucket"`
	S3Region      string `json:"s3_region"`
	S3AccessKey   string `gorm:"type:text" json:"-"`
	S3SecretKey   string `gorm:"type:text" json:"-"`
	RetentionDays int    `json:"retention_days"`
}

func DefaultImageStudioConfig() ImageStudioConfig {
	return ImageStudioConfig{
		ID: 1, Models: "[]", Agreement: DefaultImageStudioAgreement,
		Privacy: DefaultImageStudioPrivacy, StorageMode: "local",
		LocalDir: "data/image-studio", RetentionDays: 30,
	}
}

func GetImageStudioConfig() (ImageStudioConfig, error) {
	config := DefaultImageStudioConfig()
	if DB == nil {
		return config, errors.New("database unavailable")
	}
	var stored ImageStudioConfig
	err := DB.First(&stored, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return config, nil
	}
	return stored, err
}

func (c ImageStudioConfig) EnabledModels() []ImageStudioModel {
	var models []ImageStudioModel
	_ = common.Unmarshal([]byte(c.Models), &models)
	return models
}

func ImageStudioRevision(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

type ImageStudioRecord struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            int       `gorm:"index" json:"user_id"`
	Model             string    `json:"model"`
	Group             string    `json:"group"`
	Prompt            string    `gorm:"type:text" json:"prompt"`
	Size              string    `json:"size"`
	Count             int       `json:"count"`
	EstimatedPrice    float64   `json:"estimated_price"`
	ActualPrice       *float64  `json:"actual_price"`
	Status            string    `gorm:"index" json:"status"`
	Error             string    `gorm:"type:text" json:"error,omitempty"`
	AgreementRevision string    `json:"-"`
	PrivacyRevision   string    `json:"-"`
	AcceptedAt        time.Time `json:"-"`
	CreatedAt         time.Time `gorm:"index" json:"created_at"`
	ExpiresAt         time.Time `gorm:"index" json:"expires_at"`
}

type ImageStudioAsset struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	RecordID    uint   `gorm:"index" json:"-"`
	Kind        string `json:"kind"`
	Backend     string `json:"-"`
	Location    string `gorm:"type:text" json:"-"`
	StoreConfig string `gorm:"type:text" json:"-"`
	MimeType    string `json:"mime_type"`
}
