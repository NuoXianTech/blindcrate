import type {Metadata} from 'next';
import {ShopOrdersMain} from '@/components/common/admin/shop-orders';

export const metadata: Metadata = {title: '商城订单'};

export default function AdminShopOrdersPage() {
  return <ShopOrdersMain />;
}
