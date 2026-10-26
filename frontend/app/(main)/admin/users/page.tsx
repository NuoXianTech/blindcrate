import type { Metadata } from "next";
import { UsersManager } from "@/components/common/admin/UsersManager";

// AuthGuard 确认登录前不渲染页面, 校验不到, 不参与即时导航校验
export const instant = false;

export const metadata: Metadata = { title: "用户管理" };

export default function AdminUsersPage() {
  return <UsersManager />;
}
