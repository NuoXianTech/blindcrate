import {BaseService} from '../core';
import type {MatchView, MatchmakingStatus} from './types';

export class MatchService extends BaseService {
  protected static readonly basePath = '';

  /** 当前用户的匹配状态 */
  static getMatchmaking() {
    return this.get<MatchmakingStatus>('/matchmaking');
  }

  /** 加入玩法的匹配队列, 凑够一局时立即开局, 此时 match_id 不为 0 */
  static joinMatchmaking(mode: string) {
    return this.post<MatchmakingStatus>('/matchmaking', {mode});
  }

  /** 离开匹配队列, 已开局时无法离开 */
  static leaveMatchmaking() {
    return this.delete<null>('/matchmaking');
  }

  /** 当前用户参与的一局, 结束后仍可查看 */
  static getMatch(id: number) {
    return this.get<MatchView>(`/matches/${id}`);
  }
}
