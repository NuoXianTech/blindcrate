import type {TransactionActors, TransactionFilters} from '../ledger';
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

/** 钱包相关的 ws 推送 */
export const WALLET_MESSAGES = {
  /** 余额变动, data 为 BalanceChangedMessage */
  balanceChanged: 'wallet:balance_changed',
} as const;

/** 一种货币变动后的余额 */
export interface BalanceChange {
  currency: string;
  balance: number;
  frozen: number;
  /** 同一货币的流水 ID 越大越新, 只采用 ID 更大的推送 */
  transaction_id: number;
}

export interface BalanceChangedMessage {
  /** 按货币代码排序 */
  balances: BalanceChange[];
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

/** 货币定义, 对应后端 wallet.Currency */
export interface Currency {
  code: string;
  name: string;
  /** 展示用小数位, 金额统一按最小单位的整数存储 */
  decimals: number;
  sort_order: number;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 修改货币, 代码创建后不能修改; 已被使用的货币不能修改小数位, 基础货币不能停用 */
export type CurrencyRequest = Pick<Currency, 'name' | 'decimals' | 'sort_order' | 'is_enabled'>;

/** 新建货币 */
export type CreateCurrencyRequest = CurrencyRequest & Pick<Currency, 'code'>;

export interface AdjustBalanceRequest {
  currency: string;
  /** 最小单位, 正数加款, 负数扣款, 扣款后余额不能为负 */
  amount: number;
  /** 调账原因, 可以为空, 用户可在流水中看到 */
  remark: string;
  /** 每次打开表单生成一个, 重复提交只生效一次 */
  idempotency_key: string;
}

/** 全部用户的余额流水, 对应后端 admin.WalletTransactionItem */
export type AdminWalletTransaction = WalletTransaction & TransactionActors;

export interface ListAdminWalletTransactionsParams extends TransactionFilters {
  currency?: string;
}

export interface ListAdminWalletTransactionsData {
  total: number;
  transactions: AdminWalletTransaction[];
}
