import {BaseService} from '../core';
import type {ListSuspectsData, ListSuspectsParams, ListTiersData, ReviewSuspectRequest, SuspectReview, Tier, TierRequest} from './types';

export class RankedAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  /** 全部段位, 按分数线从低到高 */
  static listTiers() {
    return this.get<ListTiersData>('/ranked/tiers');
  }

  /** 分数线不能与其他段位相同 */
  static createTier(body: TierRequest) {
    return this.post<Tier>('/ranked/tiers', body);
  }

  /** 段位按段位分实时计算, 修改分数线后所有人的段位随即变化 */
  static updateTier(id: number, body: TierRequest) {
    return this.put<Tier>(`/ranked/tiers/${id}`, body);
  }

  /** 段位分不受影响, 原来在这个段位的人按剩下的分数线重新计算段位 */
  static deleteTier(id: number) {
    return this.delete<null>(`/ranked/tiers/${id}`);
  }

  /** 排位的可疑名单, 其中一人低价成交多的排在前面; 统计天数和次数见排位设置 */
  static listSuspects(params: ListSuspectsParams) {
    return this.get<ListSuspectsData>('/ranked/suspects', {params});
  }

  /** 标记为已处理, 之后两人又一起打了排位时重新进入待处理 */
  static reviewSuspect(body: ReviewSuspectRequest) {
    return this.post<SuspectReview>('/ranked/suspects/review', body);
  }
}
