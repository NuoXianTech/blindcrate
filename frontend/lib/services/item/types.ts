import type {Quality} from '../../quality';
import type {ConditionField} from '../condition';
import type {PageParams} from '../core';

/** 物品定义, 对应后端 item.Item */
export interface Item {
  id: number;
  /** 英文别名, 可以重复, 物品以 ID 区分 */
  code: string;
  name: string;
  description: string;
  /** 品质等级, 见 useQualities */
  quality: number;
  /** 类别代码, 见 ItemCategory */
  category: string;
  image_url: string;
  sell_currency: string;
  /** 官方回收价, 最小单位; 0 表示不可卖出 */
  sell_price: number;
  /** 在仓库中占的格数 */
  width: number;
  height: number;
  /** 使用效果代码, 见 ItemEffect; 空表示不能使用, 藏品没有效果 */
  effect: string;
  /** 效果参数, 结构由效果决定 */
  effect_params: Record<string, string | number | boolean>;
  /** 下架后不会掉落, 也不能再发放, 已有的保留 */
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

/** 物品类别, 由后端代码注册, 对应后端 item.Category */
export interface ItemCategory {
  code: string;
  label: string;
}

/** 管理端品质列表项, 附带使用该品质的物品数 */
export interface AdminQuality extends Quality {
  /** 使用该品质的未删除物品数, 不为 0 时不能删除 */
  item_count: number;
  created_at: string;
  updated_at: string;
}

/** 修改品质, 等级和代码创建后不能修改 */
export type QualityRequest = Pick<Quality, 'name' | 'color' | 'effect' | 'scan_ms' | 'sort_order'>;

/** 新建品质 */
export type CreateQualityRequest = QualityRequest & Pick<Quality, 'level' | 'code'>;

/** 管理端物品列表项, 附带持有人数 */
export interface AdminItem extends Item {
  holder_count: number;
}

export interface ListItemsParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  quality?: number;
  category?: string;
  is_enabled?: boolean;
}

export interface ListItemsData {
  total: number;
  items: AdminItem[];
}

/** 新建或修改物品, 修改时也需提交全部字段 */
export type ItemRequest = Pick<
  Item,
  | 'code'
  | 'name'
  | 'description'
  | 'quality'
  | 'category'
  | 'image_url'
  | 'sell_currency'
  | 'sell_price'
  | 'width'
  | 'height'
  | 'effect'
  | 'effect_params'
  | 'is_enabled'
>;

/** 物品的使用效果, 由后端代码注册, 对应后端 effect.TypeView */
export interface ItemEffect {
  code: string;
  /** 使用的场景, 如 match 对局中 */
  scene: string;
  /** 场景的展示名称, 如 "对局中" */
  scene_name: string;
  label: string;
  description: string;
  /** 参数的表单描述, 与条件类型相同 */
  fields: ConditionField[];
}
