import {Suspense} from 'react';
import type {Metadata} from 'next';
import {TransactionsMain} from '@/components/common/admin/transactions';

export const metadata: Metadata = {title: '流水查询'};

export default function AdminTransactionsPage() {
  // 读取地址栏参数, 需要 Suspense 才能静态渲染
  return (
    <Suspense>
      <TransactionsMain />
    </Suspense>
  );
}
