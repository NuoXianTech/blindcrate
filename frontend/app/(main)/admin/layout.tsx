import { AdminGuard } from "@/components/common/auth/AdminGuard";

/** 管理页面布局, 仅管理员可见; 后端接口同样会校验管理员 */
export default function AdminLayout({ children }: LayoutProps<"/admin">) {
  return <AdminGuard>{children}</AdminGuard>;
}
