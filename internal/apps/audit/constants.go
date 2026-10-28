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
	TargetUser     = "user"
	TargetSetting  = "setting" // target_id 固定为 0, 设置组见 detail.key
	TargetItem     = "item"
	TargetLootPool = "loot_pool"
	TargetMatch    = "match"
	TargetVenue    = "venue"
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
	ActionSettingsUpdate        = "admin.settings_update"
	ActionMailTest              = "admin.mail_test"
)
