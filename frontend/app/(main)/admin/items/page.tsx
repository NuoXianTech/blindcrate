import type {Metadata} from 'next';
import {ItemsMain} from '@/components/common/admin/items';

export const metadata: Metadata = {title: '物品管理'};

export default function AdminItemsPage() {
  return <ItemsMain />;
}
