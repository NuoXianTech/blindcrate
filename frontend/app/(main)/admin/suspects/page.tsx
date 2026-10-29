import type {Metadata} from 'next';
import {SuspectsMain} from '@/components/common/admin/suspects';

export const metadata: Metadata = {title: '可疑名单'};

export default function AdminSuspectsPage() {
  return <SuspectsMain />;
}
