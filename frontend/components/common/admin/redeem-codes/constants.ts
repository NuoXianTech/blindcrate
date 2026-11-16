import type {RedeemBatch, RedeemBatchRequest, RedeemBatchStatus} from '@/lib/services/redeem';

/** 列表每页条数 */
export const PAGE_SIZE = 20;

/** 兑换码对话框每页的个数 */
export const CODE_PAGE_SIZE = 20;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 与后端 redeem 的限制一致 */
export const MAX_NAME = 64;
export const MAX_BATCH_CODES = 10_000;
export const MAX_CODE_USES = 1_000_000;
export const MAX_USER_LIMIT = 10_000;
export const MAX_REWARD_LINES = 8;
export const MAX_REQUIREMENTS = 16;
/** 输入框中的兑换码最多的字符数, 含分隔用的空格和连字符 */
export const MAX_CODE_INPUT = 64;

/** 状态筛选 */
export type StatusFilter = typeof ALL | RedeemBatchStatus;

export const STATUS_LABELS: Record<RedeemBatchStatus, string> = {
  active: '生效中',
  expired: '已到期',
  disabled: '已停用',
};

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: ALL, label: '全部状态'},
  {value: 'active', label: STATUS_LABELS.active},
  {value: 'expired', label: STATUS_LABELS.expired},
  {value: 'disabled', label: STATUS_LABELS.disabled},
];

/** 批次在 now 时的状态, 与后端列表的状态筛选一致 */
export function batchStatus(batch: RedeemBatch, now: number): RedeemBatchStatus {
  if (!batch.is_enabled) return 'disabled';
  if (batch.expires_at && new Date(batch.expires_at).getTime() <= now) return 'expired';
  return 'active';
}

/** 每个兑换码可使用次数的简短说明 */
export function usesText(maxUses: number): string {
  if (maxUses === 0) return '不限次数';
  return maxUses === 1 ? '单次使用' : `每个可用 ${maxUses} 次`;
}

/** 兑换码是否还能被兑换, 不看批次的状态和到期时间 */
export function codeAvailable(used: number, maxUses: number): boolean {
  return maxUses === 0 || used < maxUses;
}

/** 规范化后的兑换码: 4-32 个字母、数字或汉字, 与后端 redeem.ValidCode 一致 */
const CODE_PATTERN = /^[\p{L}\p{Nd}]+$/u;

/** 指定的兑换码是否合法; 与后端 redeem.Normalize 一样先去掉空白和连字符, 不区分大小写 */
export function checkCode(input: string): string | null {
  const code = input.replace(/[\s-]/g, '');
  const length = [...code].length;
  return length >= 4 && length <= 32 && CODE_PATTERN.test(code) ? null : '兑换码需为 4-32 个字母、数字或汉字';
}

/** 批次转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(batch: RedeemBatch): RedeemBatchRequest {
  return {
    name: batch.name,
    reward: batch.reward,
    requirements: batch.requirements,
    max_uses: batch.max_uses,
    user_limit: batch.user_limit,
    expires_at: batch.expires_at,
    is_enabled: batch.is_enabled,
  };
}
