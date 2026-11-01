import type {Metadata} from 'next';
import {CodexMain} from '@/components/common/codex';

export const metadata: Metadata = {title: '图鉴'};

export default function CodexPage() {
  return <CodexMain />;
}
