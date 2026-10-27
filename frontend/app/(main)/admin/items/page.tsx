import type { Metadata } from "next";
import { ItemsManager } from "@/components/common/admin/ItemsManager";

export const metadata: Metadata = { title: "物品管理" };

export default function AdminItemsPage() {
  return <ItemsManager />;
}
