import type {Metadata} from 'next';
import {AuditLogsMain} from '@/components/common/admin/audit-logs';

export const metadata: Metadata = {title: '操作日志'};

export default function AdminAuditLogsPage() {
  return <AuditLogsMain />;
}
