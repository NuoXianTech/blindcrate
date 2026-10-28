import type {Metadata} from 'next';
import {LootPoolsMain} from '@/components/common/admin/loot-pools';

export const metadata: Metadata = {title: '掉落池'};

export default function AdminLootPoolsPage() {
  return <LootPoolsMain />;
}
