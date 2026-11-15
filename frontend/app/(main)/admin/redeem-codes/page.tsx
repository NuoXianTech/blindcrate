import type {Metadata} from 'next';
import {RedeemCodesMain} from '@/components/common/admin/redeem-codes';

export const metadata: Metadata = {title: '兑换码'};

export default function AdminRedeemCodesPage() {
  return <RedeemCodesMain />;
}
