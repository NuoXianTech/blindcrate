import {BaseService} from '../core';
import type {RedeemResult} from './types';

export class RedeemService extends BaseService {
  protected static readonly basePath = '/redeem';

  /** 使用兑换码, 奖励以邮件发放, 在信箱中领取; 不区分大小写, 其中的空格和连字符会被忽略 */
  static redeem(code: string) {
    return this.post<RedeemResult>('', {code});
  }
}
