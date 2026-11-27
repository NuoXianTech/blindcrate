import type {Condition} from '../condition';
import type {Bundle, RewardLine} from '../reward';
import type {PageParams} from '../core';

/** 兑换的结果, 对应后端 redeem.Result */
export interface RedeemResult {
  /** 批次的名称 */
  name: string;
  /** 邮件附件中的货币和物品, 在信箱中领取后才发放 */
  reward: RewardLine[];
}

/** 兑换码批次, 一次生成的兑换码共用这些设置, 对应后端 redeem.Batch */
export interface RedeemBatch {
  id: number;
  /** 兑换成功后展示给玩家, 也是奖励邮件的标题 */
  name: string;
  /** 兑换一次得到的奖励 */
  reward: Bundle;
  /** 使用条件, 需要全部满足 */
  requirements: Condition[];
  /** 每个兑换码可以被多少个用户兑换, 1 为单次使用, 0 表示不限 */
  max_uses: number;
  /** 每个用户最多兑换这个批次中的几个兑换码, 0 表示不限 */
  user_limit: number;
  /** 兑换码的个数, 生成后不变 */
  code_count: number;
  /** 到期时间, 不包含, 为空时不过期 */
  expires_at: string | null;
  /** 停用后不能兑换 */
  is_enabled: boolean;
  created_by: number;
  created_at: string;
  updated_at: string;
}

/** 后台列表中的批次, 对应后端 redeemadmin.BatchItem */
export interface AdminRedeemBatch extends RedeemBatch {
  /** 用户已不存在时为空 */
  created_username: string;
  /** 批次中全部兑换码已兑换的次数 */
  redeemed: number;
}

/** 批次的状态: active 生效中, expired 已到期, disabled 已停用 */
export type RedeemBatchStatus = 'active' | 'expired' | 'disabled';

export interface ListRedeemBatchesParams extends PageParams {
  /** 按名称搜索, 也可以填完整的兑换码, 查它属于哪个批次 */
  keyword?: string;
  status?: RedeemBatchStatus;
}

export interface ListRedeemBatchesData {
  total: number;
  batches: AdminRedeemBatch[];
}

/** 修改批次, 需提交全部字段; 兑换码本身不能修改 */
export type RedeemBatchRequest = Omit<RedeemBatch, 'id' | 'code_count' | 'created_by' | 'created_at' | 'updated_at'>;

/** 生成兑换码 */
export interface CreateRedeemBatchRequest extends RedeemBatchRequest {
  /** 随机生成的个数, 1-10000; 指定了兑换码时忽略 */
  count: number;
  /** 指定兑换码, 4-32 个字母、数字或汉字; 不为空时只生成这一个 */
  code: string;
  /** 重复提交只生成一次 */
  idempotency_key: string;
}

/** 一个兑换码, 对应后端 redeem.Code */
export interface RedeemCode {
  id: number;
  batch_id: number;
  code: string;
  /** 已兑换的次数 */
  used: number;
}

export interface ListRedeemCodesData {
  codes: RedeemCode[];
}

/** 兑换记录, 对应后端 redeemadmin.RecordItem */
export interface RedeemRecord {
  id: number;
  batch_id: number;
  code_id: number;
  user_id: number;
  /** 兑换时批次的奖励, 以邮件附件发给了用户 */
  reward: Bundle;
  created_at: string;
  /** 用户已不存在时为空 */
  username: string;
  /** 批次已删除时仍然返回 */
  batch_name: string;
  code: string;
}

export interface ListRedeemRecordsParams extends PageParams {
  batch_id?: number;
  user_id?: number;
  /** 完整的兑换码, 不区分大小写 */
  code?: string;
}

export interface ListRedeemRecordsData {
  total: number;
  records: RedeemRecord[];
}
