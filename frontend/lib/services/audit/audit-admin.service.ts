import {BaseService, type CodeLabel} from '../core';
import type {ListAuditLogsData, ListAuditLogsParams} from './types';

export class AuditAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  static listAuditLogs(params: ListAuditLogsParams) {
    return this.get<ListAuditLogsData>('/audit-logs', {params});
  }

  /** 操作日志的操作类型和名称, 按后端登记的顺序 */
  static listAuditActions() {
    return this.get<{ actions: CodeLabel[] }>('/audit-logs/actions');
  }
}
