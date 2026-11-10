import type {PageParams} from '../core';

/** 流水查询共用的筛选条件 */
export interface TransactionFilters extends PageParams {
  user_id?: number;
  operator_id?: number;
  biz_type?: string;
  biz_id?: string;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

/** 流水的用户和操作人, 用户已不存在或系统操作时用户名为空 */
export interface TransactionActors {
  user_id: number;
  username: string;
  operator_id: number;
  operator_username: string;
  idempotency_key: string;
}
