import type {AdminQuality, QualityRequest} from '@/lib/services/admin';
import {QUALITY_EFFECT} from '@/lib/quality';

/** 与后端 item.ValidateQuality 和请求的校验一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,32}$/;
export const COLOR_PATTERN = /^#[0-9a-f]{6}$/i;
export const MAX_LEVEL = 999;
export const MIN_SCAN_MS = 100;
export const MAX_SCAN_MS = 10_000;
export const MAX_SORT_ORDER = 1_000_000;

/** 下拉框不能用空字符串做值, 纯色用这个代替 */
export const NO_EFFECT = 'none';

/** 品质特效 */
export const EFFECTS: { value: string; label: string }[] = [
  {value: NO_EFFECT, label: '纯色'},
  {value: QUALITY_EFFECT.rainbow, label: '炫彩渐变'},
];

/** 品质转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(q: AdminQuality): QualityRequest {
  return {name: q.name, color: q.color, effect: q.effect, scan_ms: q.scan_ms, sort_order: q.sort_order};
}
