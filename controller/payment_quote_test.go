package controller

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePaymentFeeRate(t *testing.T) {
	for _, value := range []string{"0", "8", "8.5", "8.55"} {
		assert.NoError(t, operation_setting.ValidatePaymentFeeRate(value), value)
	}
	for _, value := range []string{"", "-1", "1.234", "NaN", " 8 "} {
		assert.Error(t, operation_setting.ValidatePaymentFeeRate(value), value)
	}
}

func TestQuotePaymentAddsFeeToDiscountedSubtotal(t *testing.T) {
	paymentSetting := operation_setting.GetPaymentSetting()
	originalRate := paymentSetting.PaymentFeeRate
	t.Cleanup(func() { paymentSetting.PaymentFeeRate = originalRate })
	paymentSetting.PaymentFeeRate = "8"

	quote, err := quotePayment(decimal.NewFromInt(100))
	require.NoError(t, err)
	assert.Equal(t, "100.00", quote.Subtotal.StringFixed(2))
	assert.Equal(t, "8.00", quote.Fee.StringFixed(2))
	assert.Equal(t, "108.00", quote.Total.StringFixed(2))

	quote, err = quotePayment(decimal.RequireFromString("72.50"))
	require.NoError(t, err)
	assert.Equal(t, "72.50", quote.Subtotal.StringFixed(2))
	assert.Equal(t, "78.30", quote.Total.StringFixed(2))
}

func TestQuotePaymentRoundsToTwoDecimals(t *testing.T) {
	paymentSetting := operation_setting.GetPaymentSetting()
	originalRate := paymentSetting.PaymentFeeRate
	t.Cleanup(func() { paymentSetting.PaymentFeeRate = originalRate })
	paymentSetting.PaymentFeeRate = "8.5"

	quote, err := quotePayment(decimal.RequireFromString("10.01"))
	require.NoError(t, err)
	assert.Equal(t, "10.01", quote.Subtotal.StringFixed(2))
	assert.Equal(t, "0.85", quote.Fee.StringFixed(2))
	assert.Equal(t, "10.86", quote.Total.StringFixed(2))
}

func TestQuotePaymentRejectsAmountsBeyondGatewayMinorUnits(t *testing.T) {
	paymentSetting := operation_setting.GetPaymentSetting()
	originalRate := paymentSetting.PaymentFeeRate
	t.Cleanup(func() { paymentSetting.PaymentFeeRate = originalRate })
	paymentSetting.PaymentFeeRate = "0"

	_, err := quotePaymentFloat(math.MaxFloat64)
	require.Error(t, err)
}
