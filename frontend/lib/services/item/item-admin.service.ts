import {BaseService} from '../core';
import type {
  AdminQuality,
  CreateQualityRequest,
  Item,
  ItemCategory,
  ItemEffect,
  ItemRequest,
  ListItemsData,
  ListItemsParams,
  QualityRequest,
} from './types';

export class ItemAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listItems(params: ListItemsParams) {
    return this.get<ListItemsData>('/items', {params});
  }

  static createItem(body: ItemRequest) {
    return this.post<Item>('/items', body);
  }

  static updateItem(id: number, body: ItemRequest) {
    return this.put<Item>(`/items/${id}`, body);
  }

  /** 由后端代码注册的物品类别, 按注册顺序 */
  static listItemCategories() {
    return this.get<{ categories: ItemCategory[] }>('/items/categories');
  }

  /** 由后端代码注册的使用效果和参数的字段描述, 按注册顺序 */
  static listItemEffects() {
    return this.get<{ effects: ItemEffect[] }>('/items/effects');
  }

  /** 删除后已持有的玩家背包保留为占位物品, 返回删除时的持有人数 */
  static deleteItem(id: number) {
    return this.delete<{ holder_count: number }>(`/items/${id}`);
  }

  /** 全部品质, 附带使用每个品质的物品数 */
  static listQualities() {
    return this.get<{ qualities: AdminQuality[] }>('/qualities');
  }

  static createQuality(body: CreateQualityRequest) {
    return this.post<AdminQuality>('/qualities', body);
  }

  /** 新的扫描时长只影响之后开局的对局 */
  static updateQuality(level: number, body: QualityRequest) {
    return this.put<AdminQuality>(`/qualities/${level}`, body);
  }

  /** 还有物品使用该品质时不能删除 */
  static deleteQuality(level: number) {
    return this.delete<null>(`/qualities/${level}`);
  }
}
