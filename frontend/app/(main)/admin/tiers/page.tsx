import type {Metadata} from 'next';
import {TiersMain} from '@/components/common/admin/tiers';

export const metadata: Metadata = {title: '段位管理'};

export default function AdminTiersPage() {
  return <TiersMain />;
}
