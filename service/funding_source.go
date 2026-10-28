package service

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ---------------------------------------------------------------------------
// FundingSource — 资金来源接口（钱包 or 订阅）
// ---------------------------------------------------------------------------

// FundingSource 抽象了预扣费的资金来源。
type FundingSource interface {
	// Source 返回资金来源标识："wallet" 或 "subscription"
	Source() string
	// PreConsume 从该资金来源预扣 amount 额度
	PreConsume(amount int) error
	// Settle 根据差额调整资金来源（正数补扣，负数退还）
	Settle(delta int) error
	// Refund 退还所有预扣费
	Refund() error
}

// ---------------------------------------------------------------------------
// WalletFunding — 钱包资金来源实现
// ---------------------------------------------------------------------------

// ErrInsufficientWalletQuota 钱包原子预扣失败（余额不足），未发生任何扣减。
// BillingSession 据此映射为 ErrorCodeInsufficientUserQuota，
// 使 wallet_first 等计费偏好可以回退到订阅。
var ErrInsufficientWalletQuota = errors.New("wallet quota insufficient")

type WalletFunding struct {
	userId   int
	consumed int // 实际预扣的用户额度
	sourceId string
}

func (w *WalletFunding) Source() string { return BillingSourceWallet }

func (w *WalletFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	if w.sourceId != "" {
		_, err := model.SetReferralWalletSpend(w.userId, w.sourceId, int64(w.consumed)+int64(amount), true)
		if errors.Is(err, model.ErrReferralWalletInsufficient) {
			return ErrInsufficientWalletQuota
		}
		if err == nil {
			w.consumed += amount
		}
		return err
	}
	reserved, err := model.TryReserveUserQuota(w.userId, amount)
	if err != nil {
		return err
	}
	if !reserved {
		return ErrInsufficientWalletQuota
	}
	w.consumed += amount
	return nil
}

func (w *WalletFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	if w.sourceId != "" {
		_, err := model.SetReferralWalletSpend(w.userId, w.sourceId, int64(w.consumed)+int64(delta), false)
		if err == nil {
			w.consumed += delta
		}
		return err
	}
	if delta > 0 {
		return model.DecreaseUserQuota(w.userId, delta, false)
	}
	return model.IncreaseUserQuota(w.userId, -delta, false)
}

func (w *WalletFunding) Refund() error {
	if w.consumed <= 0 {
		return nil
	}
	if w.sourceId != "" {
		_, err := model.SetReferralWalletSpend(w.userId, w.sourceId, 0, false)
		if err == nil {
			w.consumed = 0
		}
		return err
	}
	// IncreaseUserQuota 是 quota += N 的非幂等操作，不能重试，否则会多退额度。
	// 订阅的 RefundSubscriptionPreConsume 有 requestId 幂等保护所以可以重试。
	return model.IncreaseUserQuota(w.userId, w.consumed, false)
}

// ---------------------------------------------------------------------------
// SubscriptionFunding — 订阅资金来源实现
// ---------------------------------------------------------------------------

type SubscriptionFunding struct {
	requestId        string
	userId           int
	modelName        string
	amount           int64 // 预扣的订阅额度（subConsume）
	subscriptionId   int
	preConsumed      int64
	reservedExtra    int64
	deferSettlement  bool
	walletFallback   *WalletFunding
	walletSettled    bool
	reservedOnSettle bool
	// concurrencyLease 是该订阅占用的并发槽位（套餐并发限制），由 BillingSession 统一释放。
	concurrencyLease *common.ChannelConcurrencyLease
	// ownedConcurrencySubIds 是本请求已持有并发槽的订阅集合（Token 包路径获取），
	// 额度预扣对这些订阅不再重复获取，避免同一请求自我争抢槽位。
	ownedConcurrencySubIds map[int]struct{}
	// 以下字段在 PreConsume 成功后填充，供 RelayInfo 同步使用
	AmountTotal     int64
	AmountUsedAfter int64
	PlanId          int
	PlanTitle       string
}

func (s *SubscriptionFunding) Source() string {
	if s.walletSettled {
		return BillingSourceWallet
	}
	return BillingSourceSubscription
}

func (s *SubscriptionFunding) PreConsume(_ int) error {
	// amount 参数被忽略，使用内部 s.amount（已在构造时根据 preConsumedQuota 计算）
	if s.amount <= 0 {
		return nil
	}
	res, err := model.PreConsumeUserSubscription(s.requestId, s.userId, s.modelName, 0, s.amount, s.ownedConcurrencySubIds)
	if err != nil {
		return err
	}
	s.subscriptionId = res.UserSubscriptionId
	s.preConsumed = res.PreConsumed
	s.reservedExtra = 0
	s.AmountTotal = res.AmountTotal
	s.AmountUsedAfter = res.AmountUsedAfter
	if len(res.ConcurrencyLeases) > 0 {
		s.concurrencyLease = res.ConcurrencyLeases[0]
	}
	// 获取订阅计划信息
	if planInfo, err := model.GetSubscriptionPlanInfoByUserSubscriptionId(res.UserSubscriptionId); err == nil && planInfo != nil {
		s.PlanId = planInfo.PlanId
		s.PlanTitle = planInfo.PlanTitle
	}
	return nil
}

func (s *SubscriptionFunding) Settle(delta int) error {
	targetAmount := max(s.preConsumed+s.reservedExtra+int64(delta), 0)
	if s.reservedOnSettle && s.subscriptionId > 0 {
		targetAmount = s.preConsumed
	}
	if s.subscriptionId <= 0 && targetAmount > 0 {
		res, err := model.PreConsumeUserSubscription(
			s.requestId,
			s.userId,
			s.modelName,
			0,
			targetAmount,
			s.ownedConcurrencySubIds,
		)
		if err != nil {
			if s.walletFallback == nil ||
				!errors.Is(err, model.ErrNoActiveSubscription) &&
					!errors.Is(err, model.ErrSubscriptionQuotaInsufficient) &&
					!errors.Is(err, model.ErrSubscriptionConcurrencyFull) {
				return err
			}
			if err := s.walletFallback.Settle(int(targetAmount)); err != nil {
				return err
			}
			s.walletSettled = true
			s.preConsumed = targetAmount
			s.reservedExtra = 0
			return nil
		}
		s.subscriptionId = res.UserSubscriptionId
		s.preConsumed = res.PreConsumed
		s.reservedExtra = 0
		s.reservedOnSettle = true
		s.AmountTotal = res.AmountTotal
		s.AmountUsedAfter = res.AmountUsedAfter
		if len(res.ConcurrencyLeases) > 0 {
			s.concurrencyLease = res.ConcurrencyLeases[0]
		}
		if planInfo, err := model.GetSubscriptionPlanInfoByUserSubscriptionId(res.UserSubscriptionId); err == nil && planInfo != nil {
			s.PlanId = planInfo.PlanId
			s.PlanTitle = planInfo.PlanTitle
		}
		targetAmount = s.preConsumed
	}
	if s.subscriptionId <= 0 {
		return nil
	}
	if err := model.UpdateSubscriptionPreConsume(s.requestId, targetAmount, !s.deferSettlement); err != nil {
		return err
	}
	s.preConsumed = targetAmount
	s.reservedExtra = 0
	return nil
}

func (s *SubscriptionFunding) AdjustReservation(delta int) error {
	if s.subscriptionId <= 0 {
		return nil
	}
	targetAmount := max(s.preConsumed+s.reservedExtra+int64(delta), 0)
	if err := model.UpdateSubscriptionPreConsume(s.requestId, targetAmount, false); err != nil {
		return err
	}
	s.reservedExtra += int64(delta)
	return nil
}

func (s *SubscriptionFunding) Refund() error {
	if s.subscriptionId <= 0 || s.requestId == "" {
		return nil
	}
	return refundWithRetry(func() error {
		return model.RefundSubscriptionPreConsume(s.requestId)
	})
}

// refundWithRetry 尝试多次执行退款操作以提高成功率，只能用于基于事务的退款函数！！！！！！
// try to refund with retries, only for refund functions based on transactions!!!
func refundWithRetry(fn func() error) error {
	if fn == nil {
		return nil
	}
	const maxAttempts = 3
	var lastErr error
	for i := range maxAttempts {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i < maxAttempts-1 {
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
	}
	return lastErr
}
