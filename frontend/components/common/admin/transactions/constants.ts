/** 流水每页条数 */
export const PAGE_SIZE = 20;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 数字 ID 筛选, 空表示不筛选 */
export type IDFilters = Record<string, number | undefined>;

/** 解析数字 ID, 不是纯数字时返回 undefined */
export function parseID(value: string | null | undefined) {
  return value && /^\d+$/.test(value) ? Number(value) : undefined;
}
