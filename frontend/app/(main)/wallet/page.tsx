import type { Metadata } from "next";
import { WalletView } from "@/components/common/wallet/WalletView";

export const metadata: Metadata = { title: "钱包" };

export default function WalletPage() {
  return <WalletView />;
}
