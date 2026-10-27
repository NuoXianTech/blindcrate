'use client';

import {useEffect, useState} from 'react';
import {toast} from 'sonner';
import {Skeleton} from '@/components/ui/skeleton';
import {useAuth} from '@/components/common/auth';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {BalanceItem} from '@/lib/services/wallet';
import {BalanceCard} from './BalanceCard';
import {TransactionList} from './TransactionList';

/**
 * 钱包: 各货币余额 + 余额流水
 */
export function WalletMain() {
  const {user} = useAuth();
  const [balances, setBalances] = useState<BalanceItem[] | null>(null);

  useEffect(() => {
    if (!user) return;
    services.wallet.listBalances()
        .then((data) => setBalances(data.balances))
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [user]);

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-bold">钱包</h1>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {balances ?
          balances.map((b) => <BalanceCard key={b.currency} balance={b} />) :
          Array.from({length: 2}, (_, i) => <Skeleton key={i} className="h-28 rounded-xl" />)}
      </div>

      <TransactionList balances={balances ?? []} />
    </div>
  );
}
