import type {Metadata} from 'next';
import {MissionsMain} from '@/components/common/admin/missions';

export const metadata: Metadata = {title: '任务管理'};

export default function AdminMissionsPage() {
  return <MissionsMain />;
}
