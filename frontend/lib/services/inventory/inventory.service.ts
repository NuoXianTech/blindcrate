import { BaseService } from "../core";
import type {
  ListInventoryTransactionsData,
  ListInventoryTransactionsParams,
  ListUserItemsData,
  ListUserItemsParams,
} from "./types";

export class InventoryService extends BaseService {
  protected static readonly basePath = "/inventory";

  /** 当前用户背包中的物品, 按品质从高到低排序 */
  static listItems(params: ListUserItemsParams) {
    return this.get<ListUserItemsData>("/items", { params });
  }

  static listTransactions(params: ListInventoryTransactionsParams) {
    return this.get<ListInventoryTransactionsData>("/transactions", { params });
  }
}
