package operation_setting

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/shopspring/decimal"
)

const TopUpGroupUpgradeRulesOptionKey = "payment_setting.topup_group_upgrade_rules"

const (
	TopUpGroupUpgradeRuleTypeSingle     = "single"
	TopUpGroupUpgradeRuleTypeCumulative = "cumulative"
)

const maxTopUpGroupUpgradeRules = 100

var topUpGroupUpgradeAmountPattern = regexp.MustCompile(`^\d+(?:\.\d{1,2})?$`)

type TopUpGroupUpgradeRule struct {
	Type   string `json:"type"`
	Amount string `json:"amount"`
	Group  string `json:"group"`
}

func (rule TopUpGroupUpgradeRule) Threshold() (decimal.Decimal, error) {
	return decimal.NewFromString(rule.Amount)
}

type PaymentSetting struct {
	AmountOptions          []int                   `json:"amount_options"`
	AmountDiscount         map[int]float64         `json:"amount_discount"` // 充值金额对应的折扣，例如 100 元 0.9 表示 100 元充值享受 9 折优惠
	TopUpGroupUpgradeRules []TopUpGroupUpgradeRule `json:"topup_group_upgrade_rules"`

	ComplianceConfirmed    bool   `json:"compliance_confirmed"`
	ComplianceTermsVersion string `json:"compliance_terms_version"`
	ComplianceConfirmedAt  int64  `json:"compliance_confirmed_at"`
	ComplianceConfirmedBy  int    `json:"compliance_confirmed_by"`
	ComplianceConfirmedIP  string `json:"compliance_confirmed_ip"`
}

const CurrentComplianceTermsVersion = "v1"

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:          []int{10, 20, 50, 100, 200, 500},
	AmountDiscount:         map[int]float64{},
	TopUpGroupUpgradeRules: []TopUpGroupUpgradeRule{},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}

func GetTopUpGroupUpgradeRules() []TopUpGroupUpgradeRule {
	return append([]TopUpGroupUpgradeRule(nil), paymentSetting.TopUpGroupUpgradeRules...)
}

func ParseTopUpGroupUpgradeRules(value string) ([]TopUpGroupUpgradeRule, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" || trimmedValue[0] != '[' {
		return nil, fmt.Errorf("top-up group upgrade rules must be a JSON array")
	}
	var rules []TopUpGroupUpgradeRule
	if err := common.UnmarshalJsonStr(trimmedValue, &rules); err != nil {
		return nil, fmt.Errorf("top-up group upgrade rules must be a JSON array: %w", err)
	}
	if len(rules) > maxTopUpGroupUpgradeRules {
		return nil, fmt.Errorf("top-up group upgrade rules cannot exceed %d entries", maxTopUpGroupUpgradeRules)
	}

	for index, rule := range rules {
		if rule.Type != TopUpGroupUpgradeRuleTypeSingle && rule.Type != TopUpGroupUpgradeRuleTypeCumulative {
			return nil, fmt.Errorf("top-up group upgrade rule %d has an invalid type", index+1)
		}
		if rule.Amount != strings.TrimSpace(rule.Amount) ||
			!topUpGroupUpgradeAmountPattern.MatchString(rule.Amount) {
			return nil, fmt.Errorf("top-up group upgrade rule %d must have a positive amount with up to two decimal places", index+1)
		}
		threshold, err := rule.Threshold()
		if err != nil || !threshold.GreaterThan(decimal.Zero) {
			return nil, fmt.Errorf("top-up group upgrade rule %d must have a positive amount with up to two decimal places", index+1)
		}
		if rule.Group != strings.TrimSpace(rule.Group) || rule.Group == "" || len(rule.Group) > 64 {
			return nil, fmt.Errorf("top-up group upgrade rule %d has an invalid target group", index+1)
		}
	}

	return rules, nil
}

func IsPaymentComplianceConfirmed() bool {
	return paymentSetting.ComplianceConfirmed &&
		paymentSetting.ComplianceTermsVersion == CurrentComplianceTermsVersion
}
