'use client';

import {useEffect, useState} from 'react';
import {Backpack, HandCoins, ScrollText} from 'lucide-react';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import type {UserItem} from '@/lib/services/inventory';
import type {BalanceItem} from '@/lib/services/wallet';
import {ItemGrid} from './ItemGrid';
import {RecordList} from './RecordList';
import {SellOrderList} from './SellOrderList';
import type {ItemFilter} from './constants';

/**
 * 背包: 物品网格 + 背包记录 + 出售记录
 */
export function InventoryMain() {
  const {user} = useAuth();
  const [tab, setTab] = useState('items');
  const [itemFilter, setItemFilter] = useState<ItemFilter | null>(null);
  // 货币名称和小数位, 用于展示售价
  const [currencies, setCurrencies] = useState<BalanceItem[]>([]);

  useEffect(() => {
    if (!user) return;
    services.wallet.listBalances()
        .then((data) => setCurrencies(data.balances))
        .catch(() => {
        // 只影响售价的货币名称, 失败时显示货币代码
        });
  }, [user]);

  const viewRecords = (item: UserItem) => {
    setItemFilter({id: item.item_id, name: item.name});
    setTab('records');
  };

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">背包</h1>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="items">
            <Backpack />
            物品
          </TabsTrigger>
          <TabsTrigger value="records">
            <ScrollText />
            记录
          </TabsTrigger>
          <TabsTrigger value="sell-orders">
            <HandCoins />
            出售记录
          </TabsTrigger>
        </TabsList>
        <TabsContent value="items" className="mt-2">
          <ItemGrid currencies={currencies} onViewRecords={viewRecords} />
        </TabsContent>
        <TabsContent value="records" className="mt-2">
          <RecordList itemFilter={itemFilter} onClearFilter={() => setItemFilter(null)} />
        </TabsContent>
        <TabsContent value="sell-orders" className="mt-2">
          <SellOrderList currencies={currencies} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
