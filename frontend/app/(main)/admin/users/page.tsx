import type { Metadata } from "next";
import { UsersManager } from "@/components/common/admin/UsersManager";

export const metadata: Metadata = { title: "用户管理" };

export default function AdminUsersPage() {
  return <UsersManager />;
}
