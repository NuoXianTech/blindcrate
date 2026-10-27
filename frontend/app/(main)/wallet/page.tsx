import type {Metadata} from 'next';
import {WalletMain} from '@/components/common/wallet';

export const metadata: Metadata = {title: '钱包'};

export default function WalletPage() {
  return <WalletMain />;
}
