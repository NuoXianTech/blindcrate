import {AdminGuard} from '@/components/common/auth';
import {AdminNav} from '@/components/common/layout';

/** 管理页面布局, 仅管理员可见; 后端接口同样会校验管理员; 页面入口见 lib/nav.ts */
export default function AdminLayout({children}: LayoutProps<'/admin'>) {
  return (
    <AdminGuard>
      <AdminNav />
      {children}
    </AdminGuard>
  );
}
