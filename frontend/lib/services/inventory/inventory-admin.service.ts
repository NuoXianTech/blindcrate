import {BaseService} from '../core';
import type {
  AdjustInventoryRequest,
  InventoryTransaction,
  ListAdminInventoryTransactionsData,
  ListAdminInventoryTransactionsParams,
  ListUserItemsData,
  ListUserItemsParams,
} from './types';

export class InventoryAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listUserInventory(id: number, params: ListUserItemsParams) {
    return this.get<ListUserItemsData>(`/users/${id}/inventory/items`, {params});
  }

  static adjustUserInventory(id: number, body: AdjustInventoryRequest) {
    return this.post<InventoryTransaction>(`/users/${id}/inventory/adjust`, body);
  }

  static listInventoryTransactions(params: ListAdminInventoryTransactionsParams) {
    return this.get<ListAdminInventoryTransactionsData>('/inventory/transactions', {params});
  }
}
