import type { Metadata } from "next";
import { AuditLogsManager } from "@/components/common/admin/AuditLogsManager";

export const metadata: Metadata = { title: "操作日志" };

export default function AdminAuditLogsPage() {
  return <AuditLogsManager />;
}
