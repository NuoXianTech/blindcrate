import {BaseService} from '../core';
import type {CheckinResult, CheckinStatus} from './types';

export class CheckinService extends BaseService {
  protected static readonly basePath = '/checkin';

  /** 今天是否已签到、连续和累计天数、整个周期的奖励 */
  static getStatus() {
    return this.get<CheckinStatus>('');
  }

  /** 今天签到, 按连续天数领取对应一天的奖励 */
  static checkin() {
    return this.post<CheckinResult>('');
  }
}
