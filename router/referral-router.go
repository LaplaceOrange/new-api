package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func registerReferralRoutes(apiRouter *gin.RouterGroup) {
	self := apiRouter.Group("/referrals")
	self.Use(middleware.UserAuth(), middleware.DisableCache())
	{
		self.GET("/rewards", controller.GetReferralRewards)
		self.POST("/withdraw", middleware.UserCriticalRateLimit("referral-withdraw"),
			middleware.SessionCookieOriginGuard(), controller.TransferReferralRewards)
	}

	admin := apiRouter.Group("/referrals/admin")
	admin.Use(middleware.AdminAuth(), middleware.DisableCache())
	{
		admin.GET("", controller.GetReferralAdmin)
		admin.POST("/campaigns", middleware.SessionCookieOriginGuard(), controller.CreateReferralCampaign)
		admin.PUT("/campaigns/:id", middleware.SessionCookieOriginGuard(), controller.UpdateReferralCampaign)
		admin.PUT("/policy", middleware.SessionCookieOriginGuard(), controller.UpdateReferralPolicy)
		admin.GET("/withdrawals", controller.GetReferralWithdrawalStats)
	}
}
