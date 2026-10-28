import type {Metadata} from 'next';
import {MatchHistoryMain} from '@/components/common/history';

export const metadata: Metadata = {title: '战绩'};

export default function MatchesPage() {
  return <MatchHistoryMain />;
}
