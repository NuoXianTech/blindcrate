import {BaseService, captchaConfig} from '../core';
import type {CreateShopOrderRequest, ListShopItemsData, ListShopOrdersData, ListShopOrdersParams, ShopOrder} from './types';

export class ShopService extends BaseService {
  protected static readonly basePath = '/shop';

  /** 上架中的商品, 按排序值排列, 带购买条件的达成情况和已购买的份数 */
  static listItems() {
    return this.get<ListShopItemsData>('/items');
  }

  /** 扣除价格并发放商品, 全部成功或全部失败; 幂等键相同的重复提交返回原订单; captcha 为人机验证的令牌, 商店购买开启人机验证时必填 */
  static buy(body: CreateShopOrderRequest, captcha?: string) {
    return this.post<ShopOrder>('/orders', body, captchaConfig(captcha));
  }

  /** 购买记录, 从新到旧 */
  static listOrders(params: ListShopOrdersParams) {
    return this.get<ListShopOrdersData>('/orders', {params});
  }
}
