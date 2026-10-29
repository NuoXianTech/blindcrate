import type {Metadata} from 'next';
import {ShopMain} from '@/components/common/shop';

export const metadata: Metadata = {title: '商城'};

export default function ShopPage() {
  return <ShopMain />;
}
