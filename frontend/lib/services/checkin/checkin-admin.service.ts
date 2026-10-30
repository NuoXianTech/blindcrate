import {BaseService} from '../core';
import type {CheckinDaysData, UpdateCheckinDaysRequest} from './types';

/** 签到的后台接口 */
export class CheckinAdminService extends BaseService {
  protected static readonly basePath = '/admin/checkin';

  static listDays() {
    return this.get<CheckinDaysData>('/days');
  }

  /** 整体保存, 天数减少时多出的天数删除 */
  static updateDays(body: UpdateCheckinDaysRequest) {
    return this.put<CheckinDaysData>('/days', body);
  }
}
