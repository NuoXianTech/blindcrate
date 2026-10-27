import type {Metadata} from 'next';
import {InventoryMain} from '@/components/common/inventory';

export const metadata: Metadata = {title: '背包'};

export default function InventoryPage() {
  return <InventoryMain />;
}
