import type {Metadata} from 'next';
import {SeasonsMain} from '@/components/common/admin/seasons';

export const metadata: Metadata = {title: '赛季管理'};

export default function AdminSeasonsPage() {
  return <SeasonsMain />;
}
