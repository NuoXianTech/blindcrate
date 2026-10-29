import type {Bundle} from '../auction';
import type {PageParams} from '../core';
import type {RequirementProgress} from '../venue';

/** 订单的状态, 对应后端 shop.StatusPaid 等 */
export const SHOP_ORDER_STATUS = {
  paid: 'paid',
  refunded: 'refunded',
} as const;

export type ShopOrderStatus = (typeof SHOP_ORDER_STATUS)[keyof typeof SHOP_ORDER_STATUS];

/** 限购的类型, 对应后端 shop.LimitTotal 等 */
export type ShopLimitType = 'total' | 'daily' | 'season';

/** 价格或商品中的一种货币或物品, 对应后端 reward.Line */
export interface ShopLine {
  /** 货币代码, 与 item_id 二选一 */
  currency?: string;
  item_id?: number;
  name: string;
  image_url: string;
  /** 物品的品质等级, 货币为 0 */
  quality: number;
  /** 货币的展示用小数位, 物品为 0 */
  decimals: number;
  /** 货币金额或物品数量 */
  amount: number;
}

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
  goods: ShopLine[];
  /** 一份的价格, 为空时免费 */
  price: ShopLine[];
  /** 原价, 为空时不展示 */
  original_price: ShopLine[];
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
  price: ShopLine[];
  /** 实际得到的全部商品 */
  goods: ShopLine[];
  status: ShopOrderStatus;
  refunded_at: string | null;
  created_at: string;
}

export type ListShopOrdersParams = PageParams;

export interface ListShopOrdersData {
  total: number;
  orders: ShopOrder[];
}
