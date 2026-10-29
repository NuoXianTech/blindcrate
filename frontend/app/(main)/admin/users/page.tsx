import {Suspense} from 'react';
import type {Metadata} from 'next';
import {UsersMain} from '@/components/common/admin/users';

export const metadata: Metadata = {title: '用户管理'};

export default function AdminUsersPage() {
  // 读取地址栏参数, 需要 Suspense 才能静态渲染
  return (
    <Suspense>
      <UsersMain />
    </Suspense>
  );
}
