import type {Metadata} from 'next';
import {LeaderboardsMain} from '@/components/common/admin/leaderboards';

export const metadata: Metadata = {title: '榜单管理'};

export default function AdminLeaderboardsPage() {
  return <LeaderboardsMain />;
}
