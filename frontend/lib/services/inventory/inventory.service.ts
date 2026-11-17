import {BaseService, type CodeLabel} from '../core';
import type {
  ItemUseResult,
  ItemUseView,
  ListInventoryTransactionsData,
  ListInventoryTransactionsParams,
  ListSellOrdersData,
  ListSellOrdersParams,
  ListUserItemsData,
  ListUserItemsParams,
  SellItemsRequest,
  SellOrder,
  UseItemRequest,
} from './types';

export class InventoryService extends BaseService {
  protected static readonly basePath = '/inventory';

  /** 当前用户背包中的物品, 按品质从高到低排序 */
  static listItems(params: ListUserItemsParams) {
    return this.get<ListUserItemsData>('/items', {params});
  }

  /** 一个可以在背包中使用的物品、自己持有的数量和使用后可以得到的内容; 物品不能在背包中使用或暂时无法使用时失败 */
  static getUse(itemID: number) {
    return this.get<ItemUseView>(`/items/${itemID}/use`);
  }

  /** 使用一个物品, 结果在返回前已经结算; 幂等键相同的重复提交只使用一次, 返回原来的结果 */
  static use(itemID: number, body: UseItemRequest) {
    return this.post<ItemUseResult>(`/items/${itemID}/use`, body);
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
