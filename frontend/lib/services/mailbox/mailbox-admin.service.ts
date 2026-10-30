import {BaseService} from '../core';
import type {
  ListAdminMailsData,
  ListAdminMailsParams,
  ListMailBroadcastsData,
  ListMailBroadcastsParams,
  ListMailSourcesData,
  MailBroadcast,
  MailBroadcastRequest,
  RevokeMailBroadcastData,
  SendMailsData,
  SendMailsRequest,
} from './types';

/** 游戏邮件的后台接口 */
export class MailboxAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  /** 查询邮件, 从新到旧, 包括已删除的 */
  static listMails(params: ListAdminMailsParams) {
    return this.get<ListAdminMailsData>('/mails', {params});
  }

  /** 邮件来源, 用于筛选和展示 */
  static listSources() {
    return this.get<ListMailSourcesData>('/mails/sources');
  }

  /** 发给指定用户, 幂等键相同的重复提交只发送一次 */
  static sendMails(body: SendMailsRequest) {
    return this.post<SendMailsData>('/mails', body);
  }

  /** 全服邮件, 从新到旧 */
  static listBroadcasts(params: ListMailBroadcastsParams) {
    return this.get<ListMailBroadcastsData>('/mail-broadcasts', {params});
  }

  /** 发送全服邮件, 发送后不能修改 */
  static createBroadcast(body: MailBroadcastRequest) {
    return this.post<MailBroadcast>('/mail-broadcasts', body);
  }

  /** 撤回全服邮件, 附件未领取的邮件移出信箱 */
  static revokeBroadcast(id: number) {
    return this.post<RevokeMailBroadcastData>(`/mail-broadcasts/${id}/revoke`);
  }
}
