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
	"github.com/NuoXianTech/blindcrate/internal/apps/auction/auctionadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/audit/auditadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth"
	"github.com/NuoXianTech/blindcrate/internal/apps/auth/authadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/condition/conditionadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/health"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory"
	"github.com/NuoXianTech/blindcrate/internal/apps/inventory/inventoryadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/item"
	"github.com/NuoXianTech/blindcrate/internal/apps/item/itemadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard"
	"github.com/NuoXianTech/blindcrate/internal/apps/leaderboard/leaderboardadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot"
	"github.com/NuoXianTech/blindcrate/internal/apps/loot/lootadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/mailbox"
	"github.com/NuoXianTech/blindcrate/internal/apps/mailbox/mailboxadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/match"
	"github.com/NuoXianTech/blindcrate/internal/apps/match/matchadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/mission"
	"github.com/NuoXianTech/blindcrate/internal/apps/mission/missionadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/pack"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked"
	"github.com/NuoXianTech/blindcrate/internal/apps/ranked/rankedadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/season"
	"github.com/NuoXianTech/blindcrate/internal/apps/season/seasonadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings"
	"github.com/NuoXianTech/blindcrate/internal/apps/settings/settingsadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop"
	"github.com/NuoXianTech/blindcrate/internal/apps/shop/shopadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/stats"
	"github.com/NuoXianTech/blindcrate/internal/apps/upload"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue"
	"github.com/NuoXianTech/blindcrate/internal/apps/venue/venueadmin"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet"
	"github.com/NuoXianTech/blindcrate/internal/apps/wallet/walletadmin"
	"github.com/NuoXianTech/blindcrate/internal/config"
	"github.com/NuoXianTech/blindcrate/internal/otel_trace"
	"github.com/NuoXianTech/blindcrate/internal/ratelimit"
	"github.com/NuoXianTech/blindcrate/internal/storage"
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

	// 本地存储的上传文件, 访问前缀为站内路径时由 API 提供访问, 不需要会话
	if _, err := storage.Default(); err != nil {
		log.Fatalf("[API] init storage failed: %v\n", err)
	}
	if dir, prefix, ok := storage.Local(); ok {
		r.GET(prefix+"/*key", storage.LocalHandler(dir))
		r.HEAD(prefix+"/*key", storage.LocalHandler(dir))
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
				walletRouter.GET("/biz-types", wallet.ListBizTypes)
			}

			// Inventory
			inventoryRouter := apiV1Router.Group("/inventory")
			inventoryRouter.Use(auth.LoginRequired())
			{
				inventoryRouter.GET("/items", inventory.ListItems)
				inventoryRouter.GET("/transactions", inventory.ListTransactions)
				inventoryRouter.GET("/biz-types", inventory.ListBizTypes)
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
				auctionRouter.GET("/:match_id/items", auction.ListUsableItems)
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
			rankedRouter := apiV1Router.Group("/ranked")
			rankedRouter.Use(auth.LoginRequired())
			{
				rankedRouter.GET("/rating", ranked.GetRating)
				rankedRouter.GET("/tiers", ranked.ListTiers)
			}

			shopRouter := apiV1Router.Group("/shop")
			shopRouter.Use(auth.LoginRequired())
			{
				shopRouter.GET("/items", shop.ListItems)
				shopRouter.GET("/orders", shop.ListOrders)
				shopRouter.POST("/orders", ratelimit.ByUser("shop:buy", ratelimit.PerMinute(60)), shop.CreateOrder)
			}

			// Mailbox
			mailRouter := apiV1Router.Group("/mails")
			mailRouter.Use(auth.LoginRequired())
			{
				mailRouter.GET("", mailbox.ListMails)
				mailRouter.GET("/counts", mailbox.GetCounts)
				mailRouter.POST("/read-all", mailbox.ReadAllMails)
				mailRouter.POST("/claim-all", ratelimit.ByUser("mail:claim", ratelimit.PerMinute(60)), mailbox.ClaimAllMails)
				mailRouter.POST("/:id/read", mailbox.ReadMail)
				mailRouter.POST("/:id/claim", ratelimit.ByUser("mail:claim", ratelimit.PerMinute(60)), mailbox.ClaimMail)
				mailRouter.DELETE("/:id", mailbox.DeleteMail)
			}

			// Mission
			missionRouter := apiV1Router.Group("/missions")
			missionRouter.Use(auth.LoginRequired())
			{
				missionRouter.GET("", mission.ListMissions)
				missionRouter.POST("/:code/claim", ratelimit.ByUser("mission:claim", ratelimit.PerMinute(60)), mission.ClaimMission)
			}

			// Upload
			apiV1Router.POST("/uploads", auth.LoginRequired(), ratelimit.ByUser("upload", ratelimit.PerHour(60)), upload.CreateUpload)

			// Admin
			adminRouter := apiV1Router.Group("/admin")
			adminRouter.Use(auth.LoginRequired(), admin.LoginAdminRequired())
			{
				// User
				userAdminRouter := adminRouter.Group("/users")
				{
					userAdminRouter.GET("", authadmin.ListUsers)
					userAdminRouter.PATCH("/:id", authadmin.UpdateUser)
					userAdminRouter.PUT("/:id/ban", authadmin.BanUser)
					userAdminRouter.DELETE("/:id/ban", authadmin.UnbanUser)
					userAdminRouter.PUT("/:id/disabled", authadmin.SetUserDisabled)
					userAdminRouter.GET("/:id/wallet/balances", walletadmin.ListUserBalances)
					userAdminRouter.POST("/:id/wallet/adjust", walletadmin.AdjustUserBalance)
					userAdminRouter.GET("/:id/inventory/items", inventoryadmin.ListUserInventory)
					userAdminRouter.POST("/:id/inventory/adjust", inventoryadmin.AdjustUserInventory)
				}

				// Currency
				currencyAdminRouter := adminRouter.Group("/currencies")
				{
					currencyAdminRouter.GET("", walletadmin.ListCurrencies)
					currencyAdminRouter.POST("", walletadmin.CreateCurrency)
					currencyAdminRouter.PUT("/:code", walletadmin.UpdateCurrency)
				}

				// Wallet
				walletAdminRouter := adminRouter.Group("/wallet")
				{
					walletAdminRouter.GET("/transactions", walletadmin.ListWalletTransactions)
				}

				// Item
				itemAdminRouter := adminRouter.Group("/items")
				{
					itemAdminRouter.GET("", itemadmin.ListItems)
					itemAdminRouter.GET("/categories", itemadmin.ListItemCategories)
					itemAdminRouter.GET("/effects", itemadmin.ListItemEffects)
					itemAdminRouter.POST("", itemadmin.CreateItem)
					itemAdminRouter.PUT("/:id", itemadmin.UpdateItem)
					itemAdminRouter.DELETE("/:id", itemadmin.DeleteItem)
				}

				// Quality
				qualityAdminRouter := adminRouter.Group("/qualities")
				{
					qualityAdminRouter.GET("", itemadmin.ListQualities)
					qualityAdminRouter.POST("", itemadmin.CreateQuality)
					qualityAdminRouter.PUT("/:level", itemadmin.UpdateQuality)
					qualityAdminRouter.DELETE("/:level", itemadmin.DeleteQuality)
				}

				// Loot
				lootPoolAdminRouter := adminRouter.Group("/loot-pools")
				{
					lootPoolAdminRouter.GET("", lootadmin.ListLootPools)
					lootPoolAdminRouter.POST("", lootadmin.CreateLootPool)
					lootPoolAdminRouter.PUT("/:id", lootadmin.UpdateLootPool)
					lootPoolAdminRouter.DELETE("/:id", lootadmin.DeleteLootPool)
					lootPoolAdminRouter.GET("/:id/entries", lootadmin.ListLootPoolEntries)
					lootPoolAdminRouter.PUT("/:id/entries", lootadmin.ReplaceLootPoolEntries)
				}

				// Venue
				venueAdminRouter := adminRouter.Group("/venues")
				{
					venueAdminRouter.GET("", venueadmin.ListVenues)
					venueAdminRouter.POST("", venueadmin.CreateVenue)
					venueAdminRouter.POST("/simulate", ratelimit.ByUser("admin:venue_simulate", ratelimit.PerMinute(30)), venueadmin.SimulateVenue)
					venueAdminRouter.PUT("/:id", venueadmin.UpdateVenue)
					venueAdminRouter.DELETE("/:id", venueadmin.DeleteVenue)
				}

				// Condition
				conditionAdminRouter := adminRouter.Group("/conditions")
				{
					conditionAdminRouter.GET("/types", conditionadmin.ListConditionTypes)
				}

				// Match
				matchAdminRouter := adminRouter.Group("/matches")
				{
					matchAdminRouter.GET("", matchadmin.ListMatches)
					matchAdminRouter.POST("/:id/cancel", matchadmin.CancelMatch)
				}

				// Auction
				auctionAdminRouter := adminRouter.Group("/auctions")
				{
					auctionAdminRouter.GET("/:match_id", auctionadmin.GetAuction)
				}

				// Leaderboard
				leaderboardAdminRouter := adminRouter.Group("/leaderboards")
				{
					leaderboardAdminRouter.GET("", leaderboardadmin.ListLeaderboards)
					leaderboardAdminRouter.GET("/metrics", leaderboardadmin.ListLeaderboardMetrics)
					leaderboardAdminRouter.POST("", leaderboardadmin.CreateLeaderboard)
					leaderboardAdminRouter.PUT("/:id", leaderboardadmin.UpdateLeaderboard)
					leaderboardAdminRouter.DELETE("/:id", leaderboardadmin.DeleteLeaderboard)
				}

				// Season
				seasonAdminRouter := adminRouter.Group("/seasons")
				{
					seasonAdminRouter.GET("", seasonadmin.ListSeasons)
					seasonAdminRouter.POST("", seasonadmin.CreateSeason)
					seasonAdminRouter.PUT("/:id", seasonadmin.UpdateSeason)
					seasonAdminRouter.DELETE("/:id", seasonadmin.DeleteSeason)
					seasonAdminRouter.POST("/:id/end", seasonadmin.EndSeason)
				}

				// Ranked
				rankedTierAdminRouter := adminRouter.Group("/ranked/tiers")
				{
					rankedTierAdminRouter.GET("", rankedadmin.ListRankedTiers)
					rankedTierAdminRouter.POST("", rankedadmin.CreateRankedTier)
					rankedTierAdminRouter.PUT("/:id", rankedadmin.UpdateRankedTier)
					rankedTierAdminRouter.DELETE("/:id", rankedadmin.DeleteRankedTier)
				}
				rankedSuspectAdminRouter := adminRouter.Group("/ranked/suspects")
				{
					rankedSuspectAdminRouter.GET("", rankedadmin.ListRankedSuspects)
					rankedSuspectAdminRouter.POST("/review", rankedadmin.ReviewRankedSuspect)
				}

				// Shop
				shopAdminRouter := adminRouter.Group("/shop")
				{
					shopAdminRouter.GET("/items", shopadmin.ListShopItems)
					shopAdminRouter.POST("/items", shopadmin.CreateShopItem)
					shopAdminRouter.PUT("/items/:id", shopadmin.UpdateShopItem)
					shopAdminRouter.DELETE("/items/:id", shopadmin.DeleteShopItem)
					shopAdminRouter.GET("/orders", shopadmin.ListShopOrders)
					shopAdminRouter.POST("/orders/:id/refund", shopadmin.RefundShopOrder)
				}

				// Mailbox
				mailAdminRouter := adminRouter.Group("/mails")
				{
					mailAdminRouter.GET("", mailboxadmin.ListMails)
					mailAdminRouter.GET("/sources", mailboxadmin.ListMailSources)
					mailAdminRouter.POST("", mailboxadmin.SendMails)
				}
				mailBroadcastAdminRouter := adminRouter.Group("/mail-broadcasts")
				{
					mailBroadcastAdminRouter.GET("", mailboxadmin.ListMailBroadcasts)
					mailBroadcastAdminRouter.POST("", mailboxadmin.CreateMailBroadcast)
					mailBroadcastAdminRouter.POST("/:id/revoke", mailboxadmin.RevokeMailBroadcast)
				}

				// Mission
				missionAdminRouter := adminRouter.Group("/missions")
				{
					missionAdminRouter.GET("", missionadmin.ListMissions)
					missionAdminRouter.GET("/options", missionadmin.ListMissionOptions)
					missionAdminRouter.POST("", missionadmin.CreateMission)
					missionAdminRouter.PUT("/:id", missionadmin.UpdateMission)
					missionAdminRouter.DELETE("/:id", missionadmin.DeleteMission)
				}

				// Inventory
				inventoryAdminRouter := adminRouter.Group("/inventory")
				{
					inventoryAdminRouter.GET("/transactions", inventoryadmin.ListInventoryTransactions)
				}

				// Settings
				settingsAdminRouter := adminRouter.Group("/settings")
				{
					settingsAdminRouter.GET("", settingsadmin.ListSettings)
					settingsAdminRouter.GET("/:key", settingsadmin.GetSetting)
					settingsAdminRouter.PUT("/:key", settingsadmin.UpdateSetting)
					settingsAdminRouter.GET("/:key/revisions", settingsadmin.ListSettingRevisions)
					settingsAdminRouter.GET("/:key/revisions/:version", settingsadmin.GetSettingRevision)
					settingsAdminRouter.POST("/:key/rollback", settingsadmin.RollbackSetting)
					settingsAdminRouter.POST("/:key/reset", settingsadmin.ResetSetting)
					settingsAdminRouter.POST("/mail/test", ratelimit.ByUser("admin:mail_test", ratelimit.PerMinute(5)), settingsadmin.TestMail)
				}

				// Pack
				packAdminRouter := adminRouter.Group("/packs")
				{
					packAdminRouter.GET("/sections", pack.ListPackSections)
					packAdminRouter.POST("/export", ratelimit.ByUser("admin:pack_export", ratelimit.PerMinute(10)), pack.ExportPack)
					packAdminRouter.POST("/preview", ratelimit.ByUser("admin:pack_import", ratelimit.PerMinute(20)), pack.PreviewPack)
					packAdminRouter.POST("/import", ratelimit.ByUser("admin:pack_import", ratelimit.PerMinute(20)), pack.ImportPack)
					packAdminRouter.GET("/imports", pack.ListPackImports)
					packAdminRouter.GET("/imports/:id", pack.GetPackImport)
				}

				// Audit Log
				adminRouter.GET("/audit-logs", auditadmin.ListAuditLogs)
				adminRouter.GET("/audit-logs/actions", auditadmin.ListAuditActions)
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
