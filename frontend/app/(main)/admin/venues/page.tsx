import type {Metadata} from 'next';
import {VenuesMain} from '@/components/common/admin/venues';

export const metadata: Metadata = {title: '地图管理'};

export default function AdminVenuesPage() {
  return <VenuesMain />;
}
