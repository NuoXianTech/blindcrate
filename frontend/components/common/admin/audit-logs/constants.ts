import type {AuditScope} from '@/lib/services/admin';

/** 日志每页条数 */
export const PAGE_SIZE = 20;

/** 下拉框里的"全部"选项 */
export const ALL = 'all';

/** 日志来源名称 */
export const SCOPE_LABELS: Record<AuditScope, string> = {user: '用户', admin: '管理员'};
