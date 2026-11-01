import {BaseService} from '../core';
import type {AdminCodexSet, CodexSetRequest, ListCodexSetsData, ListCodexSetsParams} from './types';

/** 图鉴的后台接口 */
export class CodexAdminService extends BaseService {
  protected static readonly basePath = '/admin/codex';

  /** 收集组列表, 按排序值和 ID 排列 */
  static listSets(params: ListCodexSetsParams) {
    return this.get<ListCodexSetsData>('/sets', {params});
  }

  static createSet(body: CodexSetRequest) {
    return this.post<AdminCodexSet>('/sets', body);
  }

  /** 修改收集组, 需提交全部字段, 代码不能修改 */
  static updateSet(id: number, body: CodexSetRequest) {
    return this.put<AdminCodexSet>(`/sets/${id}`, body);
  }

  /** 软删除, 已领取的奖励不收回 */
  static deleteSet(id: number) {
    return this.delete<null>(`/sets/${id}`);
  }
}
