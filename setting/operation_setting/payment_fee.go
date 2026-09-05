package operation_setting

import (
	"fmt"
	"regexp"
)

var paymentFeeRatePattern = regexp.MustCompile(`^(?:0|[1-9]\d*)(?:\.\d{1,2})?$`)

// ValidatePaymentFeeRate validates a non-negative percentage with at most two
// fractional digits. Keeping this validation next to the persisted setting
// prevents malformed legacy option updates from affecting payment amounts.
func ValidatePaymentFeeRate(value string) error {
	if !paymentFeeRatePattern.MatchString(value) {
		return fmt.Errorf("payment fee rate must be a non-negative number with at most two decimal places")
	}
	return nil
}
