/** 昵称列表每页条数 */
export const PAGE_SIZE = 20;

/** 启用状态筛选 */
export type StatusFilter = 'all' | 'enabled' | 'disabled';

export const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  {value: 'all', label: '全部状态'},
  {value: 'enabled', label: '已启用'},
  {value: 'disabled', label: '已停用'},
];

/** 与后端 bot 的校验一致 */
export const CODE_PATTERN = /^[a-z0-9_]{1,32}$/;
export const MAX_NAME_LENGTH = 32;
export const MAX_BATCH_NAMES = 200;
export const MAX_PERSONA_NAME_LENGTH = 32;
export const MAX_DESCRIPTION_LENGTH = 255;
