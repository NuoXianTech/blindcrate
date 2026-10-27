import type { Metadata } from "next";
import { InventoryView } from "@/components/common/inventory/InventoryView";

export const metadata: Metadata = { title: "背包" };

export default function InventoryPage() {
  return <InventoryView />;
}
