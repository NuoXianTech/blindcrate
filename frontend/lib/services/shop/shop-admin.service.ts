import {BaseService} from '../core';
import type {
  AdminShopItem,
  AdminShopOrder,
  ListAdminShopItemsData,
  ListAdminShopItemsParams,
  ListAdminShopOrdersData,
  ListAdminShopOrdersParams,
  ShopItemRequest,
} from './types';

export class ShopAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listShopItems(params: ListAdminShopItemsParams) {
    return this.get<ListAdminShopItemsData>('/shop/items', {params});
  }

  static createShopItem(body: ShopItemRequest) {
    return this.post<AdminShopItem>('/shop/items', body);
  }

  /** 已有订单不受影响 */
  static updateShopItem(id: number, body: ShopItemRequest) {
    return this.put<AdminShopItem>(`/shop/items/${id}`, body);
  }

  /** 软删除, 代码不能再被新商品使用; 已有订单仍可退款 */
  static deleteShopItem(id: number) {
    return this.delete<null>(`/shop/items/${id}`);
  }

  /** 从新到旧 */
  static listShopOrders(params: ListAdminShopOrdersParams) {
    return this.get<ListAdminShopOrdersData>('/shop/orders', {params});
  }

  /** 退还价格并收回商品, 恢复库存, 不恢复限购数量; 用户的货币或物品不足以收回商品时失败 */
  static refundShopOrder(id: number) {
    return this.post<AdminShopOrder>(`/shop/orders/${id}/refund`);
  }
}
