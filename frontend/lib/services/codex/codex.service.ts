import {BaseService} from '../core';
import type {ClaimCodexSetResult, CodexData} from './types';

export class CodexService extends BaseService {
  protected static readonly basePath = '/codex';

  /** 全部启用的藏品和收集情况, 以及启用的收集组的进度和领取情况 */
  static getCodex() {
    return this.get<CodexData>('');
  }

  /** 集齐后领取收集组的奖励, 每个收集组只能领取一次 */
  static claimSet(code: string) {
    return this.post<ClaimCodexSetResult>(`/sets/${encodeURIComponent(code)}/claim`);
  }
}
