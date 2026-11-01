import type {Metadata} from 'next';
import {CodexSetsMain} from '@/components/common/admin/codex';

export const metadata: Metadata = {title: '图鉴管理'};

export default function AdminCodexPage() {
  return <CodexSetsMain />;
}
