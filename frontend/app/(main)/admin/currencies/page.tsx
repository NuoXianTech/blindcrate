import type {Metadata} from 'next';
import {CurrenciesMain} from '@/components/common/admin/currencies';

export const metadata: Metadata = {title: '货币管理'};

export default function AdminCurrenciesPage() {
  return <CurrenciesMain />;
}
