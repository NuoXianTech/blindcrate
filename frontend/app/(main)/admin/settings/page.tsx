import type {Metadata} from 'next';
import {SettingsMain} from '@/components/common/admin/settings';

export const metadata: Metadata = {title: '系统设置'};

export default function AdminSettingsPage() {
  return <SettingsMain />;
}
