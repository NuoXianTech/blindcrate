import type {Metadata} from 'next';
import {CheckinMain} from '@/components/common/admin/checkin';

export const metadata: Metadata = {title: '签到管理'};

export default function AdminCheckinPage() {
  return <CheckinMain />;
}
