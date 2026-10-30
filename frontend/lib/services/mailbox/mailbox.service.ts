import {BaseService} from '../core';
import type {ClaimMailsResult, ListMailsData, ListMailsParams, MailCounts, ReadAllMailsData} from './types';

export class MailboxService extends BaseService {
  protected static readonly basePath = '/mails';

  /** 信箱中的邮件, 从新到旧, 不含已删除和已过期的 */
  static list(params: ListMailsParams) {
    return this.get<ListMailsData>('', {params});
  }

  /** 未读和附件未领取的封数 */
  static counts() {
    return this.get<MailCounts>('/counts');
  }

  /** 标记为已读 */
  static read(id: number) {
    return this.post<null>(`/${id}/read`);
  }

  /** 全部标记为已读, 不领取附件 */
  static readAll() {
    return this.post<ReadAllMailsData>('/read-all');
  }

  /** 领取一封邮件的附件, 同时标记为已读 */
  static claim(id: number) {
    return this.post<ClaimMailsResult>(`/${id}/claim`);
  }

  /** 一键领取全部附件, 一次最多 100 封 */
  static claimAll() {
    return this.post<ClaimMailsResult>('/claim-all');
  }

  /** 删除邮件, 附件未领取的不能删除 */
  static remove(id: number) {
    return this.delete<null>(`/${id}`);
  }
}
