import type {PackActionType, PackEntryResult, PackSection, PackStatus} from '@/lib/services/admin';

/** 导入记录每页条数 */
export const PAGE_SIZE = 20;

/** 配置包名称和作者的长度上限 */
export const MAX_NAME_LENGTH = 64;

/** 配置包文件的大小上限, 与后端 maxPackSize 一致 */
export const MAX_PACK_SIZE = 64 << 20;

/** 配置种类, 按依赖排列, 与后端 packSections 一致 */
export const SECTIONS: PackSection[] = ['currencies', 'qualities', 'items', 'loot_pools', 'venues', 'shop_items', 'leaderboards', 'settings'];

/** 配置种类的名称和说明 */
export const SECTION_LABELS: Record<PackSection, string> = {
  currencies: '货币',
  qualities: '品质',
  items: '物品',
  loot_pools: '掉落池',
  venues: '地图',
  shop_items: '商品',
  leaderboards: '榜单',
  settings: '设置',
};

/** 导出时每种配置的说明 */
export const SECTION_HINTS: Record<PackSection, string> = {
  currencies: '按代码对应',
  qualities: '按等级对应',
  items: '按 ID 对应, 带上物品图片',
  loot_pools: '按代码对应, 含条目; 已删除物品的条目不导出',
  venues: '按代码对应, 带上地图图片',
  shop_items: '按代码对应, 带上商品图片; 导入时已有商品的剩余库存不变',
  leaderboards: '按代码对应',
  settings: '游戏分类的设置组, 不含密钥字段',
};

/** 可以改名导入的配置种类, 与后端 packRenamable 一致 */
export const RENAMABLE: PackSection[] = ['loot_pools', 'venues', 'shop_items', 'leaderboards'];

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
