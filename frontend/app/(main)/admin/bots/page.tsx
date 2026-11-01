import type {Metadata} from 'next';
import {BotsMain} from '@/components/common/admin/bots';

export const metadata: Metadata = {title: '人机管理'};

export default function AdminBotsPage() {
  return <BotsMain />;
}
