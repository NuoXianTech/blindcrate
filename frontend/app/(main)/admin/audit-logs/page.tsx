import type { Metadata } from "next";
import { AuditLogsManager } from "@/components/common/admin/AuditLogsManager";

// AuthGuard 确认登录前不渲染页面, 校验不到, 不参与即时导航校验
export const instant = false;

export const metadata: Metadata = { title: "操作日志" };

export default function AdminAuditLogsPage() {
  return <AuditLogsManager />;
}
