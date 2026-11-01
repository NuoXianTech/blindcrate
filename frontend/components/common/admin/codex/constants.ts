import {COLLECTIBLE} from '@/components/common/admin/items/constants';
import type {AdminItem} from '@/lib/services/admin';
import type {AdminCodexSet, CodexSetRequest} from '@/lib/services/codex';

/** 收集组列表每页条数 */
export const PAGE_SIZE = 20;

/** 启用状态筛选 */
export type StatusFilter = 'all' | 'enabled' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  {value: 'enabled', label: '已启用'},
  {value: 'disabled', label: '已停用'},
];

/** 与后端 codex.Validate 一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,32}$/;
export const MAX_SORT_ORDER = 1_000_000;
export const MAX_BUNDLE_LINES = 8;
export const MAX_SET_ITEMS = 200;

/** 从掉落池添加藏品时可以选择的掉落池数上限 */
export const POOL_LIMIT = 100;

/** 收集组中一种物品的状态: 计入进度; 已下架, 重新上架前不计入; 已删除或不是藏品, 保存时会被拒绝 */
export type MemberStatus = 'counted' | 'disabled' | 'invalid';

export function memberStatus(item: AdminItem | undefined): MemberStatus {
  if (!item || item.category !== COLLECTIBLE) return 'invalid';
  return item.is_enabled ? 'counted' : 'disabled';
}

/** 收集组转为修改接口的请求体, 修改接口需要提交全部字段 */
export function toRequest(set: AdminCodexSet): CodexSetRequest {
  return {
    code: set.code,
    name: set.name,
    description: set.description,
    item_ids: set.item_ids,
    reward: set.reward,
    sort_order: set.sort_order,
    is_enabled: set.is_enabled,
  };
}
