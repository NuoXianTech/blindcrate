import type {Metadata} from 'next';
import {LeaderboardMain} from '@/components/common/leaderboard';

export const metadata: Metadata = {title: '排行榜'};

export default function LeaderboardPage() {
  return <LeaderboardMain />;
}
