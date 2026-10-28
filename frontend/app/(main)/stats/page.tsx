import type {Metadata} from 'next';
import {StatsMain} from '@/components/common/stats';

export const metadata: Metadata = {title: '数据'};

export default function StatsPage() {
  return <StatsMain />;
}
