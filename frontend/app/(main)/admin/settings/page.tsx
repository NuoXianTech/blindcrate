import type { Metadata } from "next";
import { SettingsManager } from "@/components/common/admin/SettingsManager";

export const metadata: Metadata = { title: "系统设置" };

export default function AdminSettingsPage() {
  return <SettingsManager />;
}
