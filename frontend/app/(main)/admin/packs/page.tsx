import type {Metadata} from 'next';
import {PacksMain} from '@/components/common/admin/packs';

export const metadata: Metadata = {title: '导入导出'};

export default function AdminPacksPage() {
  return <PacksMain />;
}
