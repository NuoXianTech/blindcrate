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

package audit

// 操作范围
const (
	ScopeUser  = "user"
	ScopeAdmin = "admin"
)

// 操作对象类型
const (
	TargetUser        = "user"
	TargetSetting     = "setting" // target_id 固定为 0, 设置组见 detail.key
	TargetItem        = "item"
	TargetLootPool    = "loot_pool"
	TargetMatch       = "match"
	TargetVenue       = "venue"
	TargetLeaderboard = "leaderboard"
	TargetQuality     = "quality"  // target_id 为品质等级
	TargetCurrency    = "currency" // target_id 固定为 0, 货币代码见 detail.code
	TargetSeason      = "season"
	TargetRankedTier  = "ranked_tier"
	TargetSuspect     = "suspect" // target_id 为较小的用户 ID, 另一个见 detail.user_b
	TargetShopItem    = "shop_item"
	TargetShopOrder   = "shop_order"
)

// 操作类型, 格式为 "模块.动作"
const (
	ActionLogin                 = "auth.login"
	ActionLoginFailed           = "auth.login_failed"
	ActionRegister              = "auth.register"
	ActionLogout                = "auth.logout"
	ActionChangePassword        = "auth.change_password"
	ActionVerifyEmail           = "auth.verify_email"
	ActionUpdateProfile         = "auth.update_profile"
	ActionUserUpdate            = "admin.user_update"
	ActionUserBan               = "admin.user_ban"
	ActionUserUnban             = "admin.user_unban"
	ActionUserDisable           = "admin.user_disable"
	ActionUserEnable            = "admin.user_enable"
	ActionWalletAdjust          = "admin.wallet_adjust"
	ActionItemCreate            = "admin.item_create"
	ActionItemUpdate            = "admin.item_update"
	ActionItemDelete            = "admin.item_delete"
	ActionInventoryAdjust       = "admin.inventory_adjust"
	ActionLootPoolCreate        = "admin.loot_pool_create"
	ActionLootPoolUpdate        = "admin.loot_pool_update"
	ActionLootPoolDelete        = "admin.loot_pool_delete"
	ActionLootPoolEntriesUpdate = "admin.loot_pool_entries_update"
	ActionVenueCreate           = "admin.venue_create"
	ActionVenueUpdate           = "admin.venue_update"
	ActionVenueDelete           = "admin.venue_delete"
	ActionMatchCancel           = "admin.match_cancel"
	ActionLeaderboardCreate     = "admin.leaderboard_create"
	ActionLeaderboardUpdate     = "admin.leaderboard_update"
	ActionLeaderboardDelete     = "admin.leaderboard_delete"
	ActionQualityCreate         = "admin.quality_create"
	ActionQualityUpdate         = "admin.quality_update"
	ActionQualityDelete         = "admin.quality_delete"
	ActionCurrencyCreate        = "admin.currency_create"
	ActionCurrencyUpdate        = "admin.currency_update"
	ActionSeasonCreate          = "admin.season_create"
	ActionSeasonUpdate          = "admin.season_update"
	ActionSeasonDelete          = "admin.season_delete"
	ActionSeasonEnd             = "admin.season_end"
	ActionRankedTierCreate      = "admin.ranked_tier_create"
	ActionRankedTierUpdate      = "admin.ranked_tier_update"
	ActionRankedTierDelete      = "admin.ranked_tier_delete"
	ActionSuspectReview         = "admin.suspect_review"
	ActionShopItemCreate        = "admin.shop_item_create"
	ActionShopItemUpdate        = "admin.shop_item_update"
	ActionShopItemDelete        = "admin.shop_item_delete"
	ActionShopOrderRefund       = "admin.shop_order_refund"
	ActionSettingsUpdate        = "admin.settings_update"
	ActionSettingsRollback      = "admin.settings_rollback"
	ActionSettingsReset         = "admin.settings_reset"
	ActionMailTest              = "admin.mail_test"
)
