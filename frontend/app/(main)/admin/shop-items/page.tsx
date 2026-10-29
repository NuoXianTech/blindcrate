import type {Metadata} from 'next';
import {ShopItemsMain} from '@/components/common/admin/shop-items';

export const metadata: Metadata = {title: '商品管理'};

export default function AdminShopItemsPage() {
  return <ShopItemsMain />;
}
