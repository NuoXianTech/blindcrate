'use client';

import {useState} from 'react';
import {useSearchParams} from 'next/navigation';
import {Backpack, Wallet} from 'lucide-react';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {InventoryTransactions} from './InventoryTransactions';
import {WalletTransactions} from './WalletTransactions';
import {parseID} from './constants';

/**
 * 流水查询: 全部用户的余额流水和背包流水, 可从用户管理带 ?tab=&user_id= 跳转过来
 */
export function TransactionsMain() {
  const searchParams = useSearchParams();
  const [tab, setTab] = useState(searchParams.get('tab') === 'inventory' ? 'inventory' : 'wallet');
  const userID = parseID(searchParams.get('user_id'));

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">流水查询</h1>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="wallet">
            <Wallet />
            余额流水
          </TabsTrigger>
          <TabsTrigger value="inventory">
            <Backpack />
            背包流水
          </TabsTrigger>
        </TabsList>
        <TabsContent value="wallet" className="mt-2">
          <WalletTransactions initialUserID={userID} />
        </TabsContent>
        <TabsContent value="inventory" className="mt-2">
          <InventoryTransactions initialUserID={userID} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
