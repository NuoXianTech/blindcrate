import type {Metadata} from 'next';
import {UsersMain} from '@/components/common/admin/users';

export const metadata: Metadata = {title: '用户管理'};

export default function AdminUsersPage() {
  return <UsersMain />;
}
