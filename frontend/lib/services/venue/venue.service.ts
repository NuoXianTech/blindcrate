import {BaseService} from '../core';
import type {Venue, VenueDetail} from './types';

export class VenueService extends BaseService {
  protected static readonly basePath = '/venues';

  /** 启用的地图, 附带当前用户对入场条件的达成情况 */
  static listVenues() {
    return this.get<{ venues: Venue[] }>('');
  }

  /** 地图详情, 附带仓库各档位掉落池的公示概率 */
  static getVenue(code: string) {
    return this.get<VenueDetail>(`/${encodeURIComponent(code)}`);
  }
}
