import {BaseService} from '../core';
import type {Venue} from './types';

export class VenueService extends BaseService {
  protected static readonly basePath = '/venues';

  /** 启用的地图, 附带当前用户对入场条件的达成情况 */
  static listVenues() {
    return this.get<{ venues: Venue[] }>('');
  }
}
