'use client';

import {useCallback, useEffect, useRef, useState} from 'react';
import Link from 'next/link';
import {usePathname} from 'next/navigation';
import {useAuth} from '@/components/common/auth';
import {useWsEvent} from '@/hooks/use-ws';
import {ROUTES} from '@/lib/access';
import services from '@/lib/services';
import {WALLET_MESSAGES, type BalanceChangedMessage, type BalanceItem} from '@/lib/services/wallet';
import {formatAmount} from '@/lib/utils';
import {CurrencyIcon} from './CurrencyIcon';

/**
 * 顶部余额栏: 显示各货币的可用余额, 点击进入钱包
 * 切换页面和回到窗口时刷新, 余额变动的推送到达时立即更新; 加载前占住高度, 避免内容跳动
 */
export function BalanceBar() {
  const {user} = useAuth();
  const pathname = usePathname();
  const [balances, setBalances] = useState<BalanceItem[] | null>(null);
  // 每种货币已采用的最新流水 ID, 推送可能乱序, 旧的推送不覆盖新的余额
  const latest = useRef<Record<string, number>>({});

  const load = useCallback(async () => {
    if (!user) return;
    try {
      setBalances((await services.wallet.listBalances()).balances);
    } catch {
      // 余额栏只是展示, 失败时保留上次的数据, 不打扰用户
    }
  }, [user]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 切换页面时刷新余额
    void load();
  }, [load, pathname]);

  // 流水 ID 是全局递增的, 换了账号要重新记录
  useEffect(() => {
    latest.current = {};
  }, [user?.id]);

  useWsEvent<BalanceChangedMessage>(WALLET_MESSAGES.balanceChanged, (msg) => {
    const changes = msg.balances.filter((c) => c.transaction_id > (latest.current[c.currency] ?? 0));
    if (!changes.length) return;
    for (const c of changes) latest.current[c.currency] = c.transaction_id;
    setBalances((prev) => prev?.map((b) => {
      const change = changes.find((c) => c.currency === b.currency);
      return change ? {...b, balance: change.balance, frozen: change.frozen} : b;
    }) ?? prev);
  });

  useEffect(() => {
    const onFocus = () => void load();
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [load]);

  return (
    <div className="flex h-8 justify-end">
      {balances && balances.length > 0 && (
        <Link
          href={ROUTES.wallet}
          className="flex items-center gap-4 rounded-full border bg-card px-4 text-sm transition-colors hover:bg-accent"
        >
          {balances.map((b) => (
            <span
              key={b.currency}
              className="flex items-center gap-1.5"
              title={`${b.name}: ${formatAmount(b.balance, b.decimals)}${b.frozen ? `, 冻结 ${formatAmount(b.frozen, b.decimals)}` : ''}`}
            >
              <CurrencyIcon currency={b.currency} className="size-4" />
              <span className="font-mono tabular-nums">{formatAmount(b.balance, b.decimals, {compact: true})}</span>
            </span>
          ))}
        </Link>
      )}
    </div>
  );
}
