import type {PackActionType, PackEntryResult, PackStatus} from '@/lib/services/admin';

/** 导入记录每页条数 */
export const PAGE_SIZE = 20;

/** 配置包名称和作者的长度上限 */
export const MAX_NAME_LENGTH = 64;

/** 配置包文件的大小上限, 与后端 maxPackSize 一致 */
export const MAX_PACK_SIZE = 64 << 20;

/** 比较结果的名称和徽标样式 */
export const STATUS_LABELS: Record<PackStatus, { label: string; variant: 'default' | 'secondary' | 'outline' | 'destructive' }> = {
  new: {label: '新增', variant: 'default'},
  changed: {label: '覆盖', variant: 'secondary'},
  same: {label: '相同', variant: 'outline'},
  conflict: {label: '冲突', variant: 'destructive'},
};

/** 处理方式的名称 */
export const ACTION_LABELS: Record<PackActionType, string> = {
  overwrite: '导入',
  skip: '跳过',
  rename: '改名导入',
};

/** 一项配置在包中的标识 */
export function entryID(e: Pick<PackEntryResult, 'section' | 'key'>): string {
  return `${e.section}:${e.key}`;
}

/** 文件大小, 按 KB 或 MB 显示 */
export function formatSize(bytes: number): string {
  if (bytes < 1 << 20) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1 << 20)).toFixed(1)} MB`;
}
