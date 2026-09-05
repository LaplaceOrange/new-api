package controller

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
)

// PaymentQuote is the server-authoritative price breakdown used by every
// hosted payment gateway. Amounts are rounded to the payment currency's
// two-decimal minor unit before an order is created.
type PaymentQuote struct {
	Subtotal decimal.Decimal `json:"subtotal"`
	Fee      decimal.Decimal `json:"fee"`
	Total    decimal.Decimal `json:"total"`
	FeeRate  decimal.Decimal `json:"fee_rate"`
}

func (q PaymentQuote) TotalFloat64() float64 { return q.Total.InexactFloat64() }

func (q PaymentQuote) APIData() map[string]any {
	return map[string]any{
		"subtotal": q.Subtotal.StringFixed(2),
		"fee":      q.Fee.StringFixed(2),
		"total":    q.Total.StringFixed(2),
		"fee_rate": q.FeeRate.StringFixed(2),
	}
}

func quotePayment(subtotal decimal.Decimal) (PaymentQuote, error) {
	if subtotal.IsNegative() {
		return PaymentQuote{}, fmt.Errorf("invalid payment subtotal")
	}
	rateText := operation_setting.GetPaymentSetting().PaymentFeeRate
	if err := operation_setting.ValidatePaymentFeeRate(rateText); err != nil {
		return PaymentQuote{}, err
	}
	rate, err := decimal.NewFromString(rateText)
	if err != nil {
		return PaymentQuote{}, fmt.Errorf("invalid payment fee rate")
	}
	subtotal = subtotal.Round(2)
	total := subtotal.Mul(decimal.NewFromInt(100).Add(rate)).Div(decimal.NewFromInt(100)).Round(2)
	if total.IsNegative() || total.GreaterThan(decimal.NewFromInt(math.MaxInt64).Div(decimal.NewFromInt(100))) {
		return PaymentQuote{}, fmt.Errorf("payment amount exceeds supported range")
	}
	return PaymentQuote{Subtotal: subtotal, Fee: total.Sub(subtotal), Total: total, FeeRate: rate}, nil
}

func quotePaymentFloat(subtotal float64) (PaymentQuote, error) {
	if math.IsNaN(subtotal) || math.IsInf(subtotal, 0) {
		return PaymentQuote{}, fmt.Errorf("invalid payment subtotal")
	}
	return quotePayment(decimal.NewFromFloat(subtotal))
}
