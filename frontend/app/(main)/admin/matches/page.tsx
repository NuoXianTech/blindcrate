import {Suspense} from 'react';
import type {Metadata} from 'next';
import {MatchesMain} from '@/components/common/admin/matches';

export const metadata: Metadata = {title: '对局管理'};

export default function AdminMatchesPage() {
  // 读取地址栏参数, 需要 Suspense 才能静态渲染
  return (
    <Suspense>
      <MatchesMain />
    </Suspense>
  );
}
