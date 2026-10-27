import type {PageParams} from '../core';

/** 某种货币的余额, 对应后端 wallet.BalanceItem */
export interface BalanceItem {
  currency: string;
  name: string;
  /** 展示用小数位, 金额统一按最小单位的整数返回 */
  decimals: number;
  is_enabled: boolean;
  /** 可用余额 */
  balance: number;
  /** 冻结余额(如竞拍押金), 不计入可用 */
  frozen: number;
}

/** 用户自己的余额流水, 对应后端 wallet.TransactionItem */
export interface WalletTransaction {
  id: number;
  currency: string;
  /** 可用余额变动, 正数入账, 负数出账 */
  amount: number;
  /** 冻结余额变动, 正数冻结, 负数解冻或扣除 */
  frozen_amount: number;
  balance_after: number;
  frozen_after: number;
  biz_type: string;
  biz_id: string;
  remark: string;
  created_at: string;
}

export interface ListWalletTransactionsParams extends PageParams {
  currency?: string;
  biz_type?: string;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

export interface ListWalletTransactionsData {
  total: number;
  transactions: WalletTransaction[];
}
