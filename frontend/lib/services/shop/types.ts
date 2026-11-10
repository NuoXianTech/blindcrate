import type {Condition} from '../condition';
import type {PageParams} from '../core';
import type {Bundle, RewardLine} from '../reward';
import type {RequirementProgress} from '../venue';

/** 订单的状态, 对应后端 shop.StatusPaid 等 */
export const SHOP_ORDER_STATUS = {
  paid: 'paid',
  refunded: 'refunded',
} as const;

export type ShopOrderStatus = (typeof SHOP_ORDER_STATUS)[keyof typeof SHOP_ORDER_STATUS];

/** 限购的类型, 对应后端 shop.LimitTotal 等 */
export type ShopLimitType = 'total' | 'daily' | 'season';

/** 一种限购和已购买的份数, 对应后端 shop.LimitView */
export interface ShopLimit {
  type: ShopLimitType;
  limit: number;
  /** 每赛季限购在没有进行中的赛季时为 0, 此时不能购买 */
  bought: number;
}

/** 上架中的商品, 对应后端 shop.ItemView */
export interface ShopItem {
  code: string;
  name: string;
  description: string;
  image_url: string;
  /** 页签, 为空时归入默认页签 */
  tab: string;
  /** 一份商品 */
  goods: RewardLine[];
  /** 一份的价格, 为空时免费 */
  price: RewardLine[];
  /** 原价, 为空时不展示 */
  original_price: RewardLine[];
  /** 每个购买条件的达成情况 */
  requirements: RequirementProgress[];
  /** 满足全部购买条件 */
  eligible: boolean;
  /** 设置了的限购 */
  limits: ShopLimit[];
  /** 全服剩余库存, 为空时不限 */
  stock: number | null;
  /** 下架时间, 为空时不限 */
  ends_at: string | null;
}

export interface ListShopItemsData {
  items: ShopItem[];
}

/** 购买商品 */
export interface CreateShopOrderRequest {
  code: string;
  /** 份数, 1-99 */
  quantity: number;
  /** 商城中看到的一份的价格, 与当前不一致时拒绝购买 */
  price: Bundle;
  /** 每次打开购买确认框生成一个, 重复提交只成交一次 */
  idempotency_key: string;
}

/** 当前用户的订单, 对应后端 shop.OrderView */
export interface ShopOrder {
  id: number;
  item_code: string;
  name: string;
  quantity: number;
  /** 实际支付的总价 */
  price: RewardLine[];
  /** 实际得到的全部商品 */
  goods: RewardLine[];
  status: ShopOrderStatus;
  refunded_at: string | null;
  created_at: string;
}

export type ListShopOrdersParams = PageParams;

export interface ListShopOrdersData {
  total: number;
  orders: ShopOrder[];
}

/** 每人限购的份数, 0 为不限 */
export interface ShopLimits {
  /** 累计 */
  total: number;
  /** 每天, 按服务器时区 */
  daily: number;
  /** 每赛季, 没有进行中的赛季时不能购买 */
  season: number;
}

/** 商品, 对应后端 shop.Item */
export interface AdminShopItem {
  id: number;
  /** 拼进限购计数的指标, 创建后不能修改 */
  code: string;
  name: string;
  description: string;
  image_url: string;
  /** 商城中的页签, 为空时归入默认页签 */
  tab: string;
  /** 商城中从小到大排列 */
  sort_order: number;
  /** 一份商品, 不能为空 */
  goods: Bundle;
  /** 一份的价格, 为空时免费 */
  price: Bundle;
  /** 原价, 只用于展示折扣 */
  original_price: Bundle;
  /** 购买条件, 需要全部满足 */
  requirements: Condition[];
  limits: ShopLimits;
  /** 全服剩余库存, 为空时不限 */
  stock: number | null;
  /** 已售出的份数, 退款时减去 */
  sold: number;
  /** 上架时间, 包含, 为空时不限 */
  starts_at: string | null;
  /** 下架时间, 不包含, 为空时不限 */
  ends_at: string | null;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListAdminShopItemsParams extends PageParams {
  /** 按代码或名称搜索, 不区分大小写 */
  keyword?: string;
  tab?: string;
  is_enabled?: boolean;
}

export interface ListAdminShopItemsData {
  total: number;
  items: AdminShopItem[];
  /** 未删除的商品用到的页签, 按名称排列, 不含空页签 */
  tabs: string[];
}

/** 新建或修改商品, 修改时也需提交全部字段, 代码不能修改; 时间精确到分钟 */
export type ShopItemRequest = Omit<AdminShopItem, 'id' | 'sold' | 'created_at' | 'updated_at'>;

/** 订单, 对应后端 admin.ShopOrderItem */
export interface AdminShopOrder {
  id: number;
  user_id: number;
  /** 用户已不存在时为空 */
  username: string;
  shop_item_id: number;
  /** 下单时的商品代码和名称 */
  item_code: string;
  name: string;
  quantity: number;
  /** 实际支付的总价 */
  price: Bundle;
  /** 实际得到的全部商品 */
  goods: Bundle;
  status: ShopOrderStatus;
  refunded_at: string | null;
  /** 退款的管理员, 没有退款时为 0 */
  refunded_by: number;
  /** 没有退款时为空 */
  refunded_username: string;
  created_at: string;
}

export interface ListAdminShopOrdersParams extends PageParams {
  user_id?: number;
  shop_item_id?: number;
  status?: ShopOrderStatus;
  /** RFC3339, 包含 */
  start_time?: string;
  /** RFC3339, 不包含 */
  end_time?: string;
}

export interface ListAdminShopOrdersData {
  total: number;
  orders: AdminShopOrder[];
}
