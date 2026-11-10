import {BaseService} from '../core';
import type {AdminMatch, ListAdminMatchesData, ListAdminMatchesParams} from './types';

export class MatchAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listMatches(params: ListAdminMatchesParams) {
    return this.get<ListAdminMatchesData>('/matches', {params});
  }

  /** 取消进行中的对局, 玩法会退还入场代价并解冻资金; 原因记录在操作日志中 */
  static cancelMatch(id: number, reason: string) {
    return this.post<Omit<AdminMatch, 'players'>>(`/matches/${id}/cancel`, {reason});
  }
}
