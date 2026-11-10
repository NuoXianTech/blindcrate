import {BaseService} from '../core';
import type {ListLootPoolsData, ListLootPoolsParams, LootEntryRequest, LootPool, LootPoolEntriesData, LootPoolRequest} from './types';

export class LootAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listLootPools(params: ListLootPoolsParams) {
    return this.get<ListLootPoolsData>('/loot-pools', {params});
  }

  static createLootPool(body: LootPoolRequest) {
    return this.post<LootPool>('/loot-pools', body);
  }

  static updateLootPool(id: number, body: LootPoolRequest) {
    return this.put<LootPool>(`/loot-pools/${id}`, body);
  }

  /** 一同删除全部条目, 返回删除的条目数 */
  static deleteLootPool(id: number) {
    return this.delete<{ entry_count: number }>(`/loot-pools/${id}`);
  }

  static listLootPoolEntries(id: number) {
    return this.get<LootPoolEntriesData>(`/loot-pools/${id}/entries`);
  }

  /** 用提交的条目整体替换, 提交空数组清空 */
  static replaceLootPoolEntries(id: number, entries: LootEntryRequest[]) {
    return this.put<LootPoolEntriesData>(`/loot-pools/${id}/entries`, {entries});
  }
}
