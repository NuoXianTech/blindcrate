import type {Metadata} from 'next';
import {MatchesMain} from '@/components/common/admin/matches';

export const metadata: Metadata = {title: '对局管理'};

export default function AdminMatchesPage() {
  return <MatchesMain />;
}
