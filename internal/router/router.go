/*
 * MIT License
 *
 * Copyright (c) 2026 nuoxi4n, NuoXianTech
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package router

import (
	"context"
	"fmt"
	"log"
	"strconv"

	_ "github.com/NuoXianTech/blindcrate/docs"
	"github.com/NuoXianTech/blindcrate/internal/apps/admin"
	"github.com/NuoXianTech/blindcrate/internal/apps/auction"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	_ "github.com/NuoXianTech/blindcrate/internal/apps/economy" // 注册时发放新手资金
	"github.com/NuoXianTech/blindcrate/internal/apps/health"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/otel_trace"
	"github.com/NuoXianTech/blindcrate/internal/ratelimit"
	"github.com/NuoXianTech/blindcrate/internal/ws"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func Serve() {
	defer otel_trace.Shutdown(context.Background())

	// 运行模式
	if config.Config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 只采信可信代理转发的 X-Forwarded-For, 否则客户端可以伪造 IP 绕过按 IP 的限流
	if err := r.SetTrustedProxies(config.Config.App.TrustedProxies); err != nil {
		log.Fatalf("[API] set trusted proxies failed: %v\n", err)
	}

	// Session
	sessionStore, err := redis.NewStoreWithDB(
		config.Config.Redis.MinIdleConn,
		"tcp",
		fmt.Sprintf("%s:%d", config.Config.Redis.Host, config.Config.Redis.Port),
		config.Config.Redis.Username,
		config.Config.Redis.Password,
		strconv.Itoa(config.Config.Redis.DB),
		[]byte(config.Config.App.SessionSecret),
	)
	if err != nil {
		log.Fatalf("[API] init session store failed: %v\n", err)
	}
	sessionStore.Options(
		sessions.Options{
			Path:     "/",
			Domain:   config.Config.App.SessionDomain,
			MaxAge:   config.Config.App.SessionAge,
			HttpOnly: config.Config.App.SessionHttpOnly,
			Secure:   config.Config.App.SessionSecure, // 若用 HTTPS 可以设 true
		},
	)
	r.Use(sessions.Sessions(config.Config.App.SessionCookieName, sessionStore))

	// 补充中间件
	r.Use(otelgin.Middleware(config.Config.App.AppName), loggerMiddleware())

	apiGroup := r.Group(config.Config.App.APIPrefix)
	{
		if config.Config.App.Env == "development" {
			// Swagger
			apiGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}

		// API V1
		apiV1Router := apiGroup.Group("/v1")
		{
			// Health
			apiV1Router.GET("/health", health.Health)
			apiV1Router.GET("/ready", health.Ready)

			// Auth
			apiV1Router.POST("/auth/login", ratelimit.ByIP("auth:login", ratelimit.PerMinute(10)), auth.Login)
			apiV1Router.POST("/auth/register", ratelimit.ByIP("auth:register", ratelimit.PerHour(10)), auth.Register)
			apiV1Router.POST("/auth/logout", auth.Logout)
			apiV1Router.GET("/auth/user-info", auth.LoginRequired(), auth.UserInfo)
			apiV1Router.PATCH("/auth/user-info", auth.LoginRequired(), auth.UpdateUserInfo)
			apiV1Router.PUT("/auth/password", auth.LoginRequired(), ratelimit.ByUser("auth:change_password", ratelimit.PerHour(10)), auth.ChangePassword)
			apiV1Router.POST("/auth/email/code", auth.LoginRequired(), ratelimit.ByUser("auth:send_email_code", ratelimit.PerHour(10)), auth.SendEmailCode)
			apiV1Router.POST("/auth/email/verify", auth.LoginRequired(), auth.VerifyEmail)

			// Settings
			apiV1Router.GET("/settings/public", settings.PublicSettings)

			// Item
			apiV1Router.GET("/qualities", item.ListQualities)

			// WebSocket
			apiV1Router.GET("/ws", auth.LoginRequired(), ws.Connect)

			// Wallet
			walletRouter := apiV1Router.Group("/wallet")
			walletRouter.Use(auth.LoginRequired())
			{
				walletRouter.GET("/balances", wallet.ListBalances)
				walletRouter.GET("/transactions", wallet.ListTransactions)
			}

			// Inventory
			inventoryRouter := apiV1Router.Group("/inventory")
			inventoryRouter.Use(auth.LoginRequired())
			{
				inventoryRouter.GET("/items", inventory.ListItems)
				inventoryRouter.GET("/transactions", inventory.ListTransactions)
				inventoryRouter.POST("/sell", inventory.SellItems)
				inventoryRouter.GET("/sell-orders", inventory.ListSellOrders)
			}

			// Loot
			apiV1Router.GET("/loot-pools/:code/odds", auth.LoginRequired(), loot.GetPoolOdds)

			// Venue
			venueRouter := apiV1Router.Group("/venues")
			venueRouter.Use(auth.LoginRequired())
			{
				venueRouter.GET("", venue.ListVenues)
				venueRouter.GET("/:code", venue.GetVenue)
			}

			// Match
			matchmakingRouter := apiV1Router.Group("/matchmaking")
			matchmakingRouter.Use(auth.LoginRequired())
			{
				matchmakingRouter.GET("", match.GetMatchmaking)
				matchmakingRouter.POST("", ratelimit.ByUser("match:join", ratelimit.PerMinute(30)), match.JoinMatchmaking)
				matchmakingRouter.DELETE("", match.LeaveMatchmaking)
			}
			matchRouter := apiV1Router.Group("/matches")
			matchRouter.Use(auth.LoginRequired())
			{
				matchRouter.GET("", match.ListMatches)
				matchRouter.GET("/current", match.GetCurrentMatch)
				matchRouter.GET("/:id", match.GetMatch)
			}

			// Auction
			auctionRouter := apiV1Router.Group("/auctions")
			auctionRouter.Use(auth.LoginRequired())
			{
				auctionRouter.GET("/:match_id", auction.GetAuction)
			}

			// Leaderboard
			leaderboardRouter := apiV1Router.Group("/leaderboards")
			leaderboardRouter.Use(auth.LoginRequired())
			{
				leaderboardRouter.GET("", leaderboard.ListBoards)
				leaderboardRouter.GET("/:code", leaderboard.GetBoard)
			}

			// Stats
			apiV1Router.GET("/stats", auth.LoginRequired(), stats.GetStats)

			// Season
			apiV1Router.GET("/seasons/current", auth.LoginRequired(), season.GetCurrent)

			// Ranked
			apiV1Router.GET("/ranked/rating", auth.LoginRequired(), ranked.GetRating)

			// Admin
			adminRouter := apiV1Router.Group("/admin")
			adminRouter.Use(auth.LoginRequired(), admin.LoginAdminRequired())
			{
				// User
				userAdminRouter := adminRouter.Group("/users")
				{
					userAdminRouter.GET("", admin.ListUsers)
					userAdminRouter.PATCH("/:id", admin.UpdateUser)
					userAdminRouter.PUT("/:id/ban", admin.BanUser)
					userAdminRouter.DELETE("/:id/ban", admin.UnbanUser)
					userAdminRouter.PUT("/:id/disabled", admin.SetUserDisabled)
					userAdminRouter.GET("/:id/wallet/balances", admin.ListUserBalances)
					userAdminRouter.POST("/:id/wallet/adjust", admin.AdjustUserBalance)
					userAdminRouter.GET("/:id/inventory", admin.ListUserInventory)
					userAdminRouter.POST("/:id/inventory/adjust", admin.AdjustUserInventory)
				}

				// Wallet
				walletAdminRouter := adminRouter.Group("/wallet")
				{
					walletAdminRouter.GET("/currencies", admin.ListCurrencies)
					walletAdminRouter.POST("/currencies", admin.CreateCurrency)
					walletAdminRouter.PUT("/currencies/:code", admin.UpdateCurrency)
					walletAdminRouter.GET("/transactions", admin.ListWalletTransactions)
				}

				// Item
				itemAdminRouter := adminRouter.Group("/items")
				{
					itemAdminRouter.GET("", admin.ListItems)
					itemAdminRouter.GET("/categories", admin.ListItemCategories)
					itemAdminRouter.POST("", admin.CreateItem)
					itemAdminRouter.PUT("/:id", admin.UpdateItem)
					itemAdminRouter.DELETE("/:id", admin.DeleteItem)
				}

				// Quality
				qualityAdminRouter := adminRouter.Group("/qualities")
				{
					qualityAdminRouter.GET("", admin.ListQualities)
					qualityAdminRouter.POST("", admin.CreateQuality)
					qualityAdminRouter.PUT("/:level", admin.UpdateQuality)
					qualityAdminRouter.DELETE("/:level", admin.DeleteQuality)
				}

				// Loot
				lootPoolAdminRouter := adminRouter.Group("/loot-pools")
				{
					lootPoolAdminRouter.GET("", admin.ListLootPools)
					lootPoolAdminRouter.POST("", admin.CreateLootPool)
					lootPoolAdminRouter.PUT("/:id", admin.UpdateLootPool)
					lootPoolAdminRouter.DELETE("/:id", admin.DeleteLootPool)
					lootPoolAdminRouter.GET("/:id/entries", admin.ListLootPoolEntries)
					lootPoolAdminRouter.PUT("/:id/entries", admin.ReplaceLootPoolEntries)
				}

				// Venue
				venueAdminRouter := adminRouter.Group("/venues")
				{
					venueAdminRouter.GET("", admin.ListVenues)
					venueAdminRouter.POST("", admin.CreateVenue)
					venueAdminRouter.POST("/simulate", ratelimit.ByUser("admin:venue_simulate", ratelimit.PerMinute(30)), admin.SimulateVenue)
					venueAdminRouter.PUT("/:id", admin.UpdateVenue)
					venueAdminRouter.DELETE("/:id", admin.DeleteVenue)
				}

				// Condition
				conditionAdminRouter := adminRouter.Group("/conditions")
				{
					conditionAdminRouter.GET("/types", admin.ListConditionTypes)
				}

				// Match
				matchAdminRouter := adminRouter.Group("/matches")
				{
					matchAdminRouter.GET("", admin.ListMatches)
					matchAdminRouter.GET("/:id/auction", admin.GetMatchAuction)
					matchAdminRouter.POST("/:id/cancel", admin.CancelMatch)
				}

				// Leaderboard
				leaderboardAdminRouter := adminRouter.Group("/leaderboards")
				{
					leaderboardAdminRouter.GET("", admin.ListLeaderboards)
					leaderboardAdminRouter.GET("/metrics", admin.ListLeaderboardMetrics)
					leaderboardAdminRouter.POST("", admin.CreateLeaderboard)
					leaderboardAdminRouter.PUT("/:id", admin.UpdateLeaderboard)
					leaderboardAdminRouter.DELETE("/:id", admin.DeleteLeaderboard)
				}

				// Season
				seasonAdminRouter := adminRouter.Group("/seasons")
				{
					seasonAdminRouter.GET("", admin.ListSeasons)
					seasonAdminRouter.POST("", admin.CreateSeason)
					seasonAdminRouter.PUT("/:id", admin.UpdateSeason)
					seasonAdminRouter.DELETE("/:id", admin.DeleteSeason)
					seasonAdminRouter.POST("/:id/end", admin.EndSeason)
				}

				// Inventory
				inventoryAdminRouter := adminRouter.Group("/inventory")
				{
					inventoryAdminRouter.GET("/transactions", admin.ListInventoryTransactions)
				}

				// Settings
				settingsAdminRouter := adminRouter.Group("/settings")
				{
					settingsAdminRouter.GET("", admin.ListSettings)
					settingsAdminRouter.GET("/:key", admin.GetSetting)
					settingsAdminRouter.PUT("/:key", admin.UpdateSetting)
					settingsAdminRouter.GET("/:key/revisions", admin.ListSettingRevisions)
					settingsAdminRouter.GET("/:key/revisions/:version", admin.GetSettingRevision)
					settingsAdminRouter.POST("/:key/rollback", admin.RollbackSetting)
					settingsAdminRouter.POST("/:key/reset", admin.ResetSetting)
					settingsAdminRouter.POST("/mail/test", ratelimit.ByUser("admin:mail_test", ratelimit.PerMinute(5)), admin.TestMail)
				}

				// Audit Log
				adminRouter.GET("/audit-logs", admin.ListAuditLogs)
			}
		}
	}

	// WebSocket 推送
	if err := ws.Start(); err != nil {
		log.Fatalf("[API] start websocket failed: %v\n", err)
	}

	// Serve
	if err := r.Run(config.Config.App.Addr); err != nil {
		log.Fatalf("[API] serve api failed: %v\n", err)
	}
}
