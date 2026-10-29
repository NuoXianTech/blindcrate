/** 余额流水的业务类型名称, 与后端 internal/apps/wallet/constants.go 的 Biz 常量对应 */
export const WALLET_BIZ_LABELS: Record<string, string> = {
  admin_adjust: '管理员调整',
  item_sell: '出售物品',
  item_sell_fee: '出售手续费',
  signup_grant: '新手资金',
  match_entry_fee: '对局入场费',
  match_carry_freeze: '冻结带入资金',
  match_carry_release: '解冻带入资金',
  match_bid_settle: '竞拍成交',
  match_refund: '对局退还',
  match_reward: '竞拍所得',
  season_reward: '赛季奖励',
};

/** 背包流水的业务类型名称, 与后端 internal/apps/inventory/constants.go 的 Biz 常量对应 */
export const INVENTORY_BIZ_LABELS: Record<string, string> = {
  admin_adjust: '管理员调整',
  item_sell: '出售物品',
  match_entry_fee: '对局入场费',
  match_refund: '对局退还',
  match_reward: '竞拍所得',
  season_reward: '赛季奖励',
};
