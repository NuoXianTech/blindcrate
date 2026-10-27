'use client';

import {useEffect} from 'react';
import {useRouter} from 'next/navigation';
import {toast} from 'sonner';
import {useAuth} from './AuthProvider';
import {ROUTES} from '@/lib/access';

/**
 * 管理员页面: 确认不是管理员时回到游戏首页
 * 确认前照常渲染页面(不挡住渲染, 页面才能参与即时导航校验), 数据由后端接口校验管理员保护
 */
export function AdminGuard({children}: { children: React.ReactNode }) {
  const {user} = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (user && !user.is_admin) {
      toast.error('这个页面只有管理员可以访问', {id: 'admin-only'});
      router.replace(ROUTES.gameHome);
    }
  }, [user, router]);

  return <>{children}</>;
}
