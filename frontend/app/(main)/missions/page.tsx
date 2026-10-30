import type {Metadata} from 'next';
import {MissionsMain} from '@/components/common/missions';

export const metadata: Metadata = {title: '任务'};

export default function MissionsPage() {
  return <MissionsMain />;
}
