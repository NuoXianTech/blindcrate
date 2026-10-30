import {BaseService, type CodeLabel} from '../core';
import type {
  ListInventoryTransactionsData,
  ListInventoryTransactionsParams,
  ListSellOrdersData,
  ListSellOrdersParams,
  ListUserItemsData,
  ListUserItemsParams,
  SellItemsRequest,
  SellOrder,
} from './types';

export class InventoryService extends BaseService {
  protected static readonly basePath = '/inventory';

  /** 当前用户背包中的物品, 按品质从高到低排序 */
  static listItems(params: ListUserItemsParams) {
    return this.get<ListUserItemsData>('/items', {params});
  }

  static listTransactions(params: ListInventoryTransactionsParams) {
    return this.get<ListInventoryTransactionsData>('/transactions', {params});
  }

  /** 按当前的卖出价出售并扣除手续费, 全部成功或全部失败; 幂等键相同的重复提交返回原出售单 */
  static sell(body: SellItemsRequest) {
    return this.post<SellOrder>('/sell', body);
  }

  static listSellOrders(params: ListSellOrdersParams) {
    return this.get<ListSellOrdersData>('/sell-orders', {params});
  }

  /** 背包流水的业务类型和名称, 按后端登记的顺序 */
  static listBizTypes() {
    return this.get<{ biz_types: CodeLabel[] }>('/biz-types');
  }
}
