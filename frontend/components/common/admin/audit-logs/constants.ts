import type {AuditScope} from '@/lib/services/admin';

/** 日志每页条数 */
export const PAGE_SIZE = 20;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 操作名称, 与后端 internal/audit 的 Action 常量对应 */
export const ACTION_LABELS: Record<string, string> = {
  'auth.login': '登录',
  'auth.login_failed': '登录失败',
  'auth.register': '注册',
  'auth.logout': '退出登录',
  'auth.change_password': '修改密码',
  'auth.verify_email': '绑定邮箱',
  'auth.update_profile': '修改资料',
  'admin.user_update': '修改用户资料',
  'admin.user_ban': '封禁用户',
  'admin.user_unban': '解除封禁',
  'admin.user_disable': '停用用户',
  'admin.user_enable': '启用用户',
  'admin.wallet_adjust': '调整余额',
  'admin.inventory_adjust': '调整背包',
  'admin.item_create': '新建物品',
  'admin.item_update': '修改物品',
  'admin.item_delete': '删除物品',
  'admin.quality_create': '新建品质',
  'admin.quality_update': '修改品质',
  'admin.quality_delete': '删除品质',
  'admin.currency_create': '新建货币',
  'admin.currency_update': '修改货币',
  'admin.loot_pool_create': '新建掉落池',
  'admin.loot_pool_update': '修改掉落池',
  'admin.loot_pool_delete': '删除掉落池',
  'admin.loot_pool_entries_update': '修改掉落池条目',
  'admin.venue_create': '新建地图',
  'admin.venue_update': '修改地图',
  'admin.venue_delete': '删除地图',
  'admin.match_cancel': '取消对局',
  'admin.leaderboard_create': '新建榜单',
  'admin.leaderboard_update': '修改榜单',
  'admin.leaderboard_delete': '删除榜单',
  'admin.season_create': '新建赛季',
  'admin.season_update': '修改赛季',
  'admin.season_delete': '删除赛季',
  'admin.season_end': '提前结束赛季',
  'admin.ranked_tier_create': '新建段位',
  'admin.ranked_tier_update': '修改段位',
  'admin.ranked_tier_delete': '删除段位',
  'admin.suspect_review': '处理可疑账号',
  'admin.shop_item_create': '新建商品',
  'admin.shop_item_update': '修改商品',
  'admin.shop_item_delete': '删除商品',
  'admin.shop_order_refund': '订单退款',
  'admin.settings_update': '修改系统设置',
  'admin.settings_rollback': '回滚系统设置',
  'admin.settings_reset': '恢复系统设置默认值',
  'admin.mail_test': '发送测试邮件',
  'admin.pack_export': '导出配置包',
  'admin.pack_import': '导入配置包',
};

/** 日志来源名称 */
export const SCOPE_LABELS: Record<AuditScope, string> = {user: '用户', admin: '管理员'};
