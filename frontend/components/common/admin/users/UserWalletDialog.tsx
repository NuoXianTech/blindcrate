'use client';

import {useCallback, useEffect, useState} from 'react';
import Link from 'next/link';
import {ScrollText, Snowflake} from 'lucide-react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {DialogFooter} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Skeleton} from '@/components/ui/skeleton';
import {SubmitButton} from '@/components/common/form';
import {CurrencyIcon} from '@/components/common/wallet';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminUser} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatAmount, parseAmount} from '@/lib/utils';
import {DirectionToggle} from './DirectionToggle';
import {UserAssetDialog} from './UserAssetDialog';
import {MAX_AMOUNT, transactionsHref, type Direction} from './constants';

/**
 * 查看和调整用户的钱包余额
 */
export function UserWalletDialog({user, onClose}: { user: AdminUser | null; onClose: () => void }) {
  return (
    <UserAssetDialog user={user} title="钱包" onClose={onClose}>
      {/* key 保证切换用户时表单重置 */}
      {user && <WalletPanel key={user.id} user={user} />}
    </UserAssetDialog>
  );
}

/**
 * 余额卡片 + 加款/扣款表单
 */
function WalletPanel({user}: { user: AdminUser }) {
  const {pending, submit} = useSubmit();
  const [balances, setBalances] = useState<BalanceItem[] | null>(null);
  const [direction, setDirection] = useState<Direction>('add');
  const [currency, setCurrency] = useState('');
  const [amount, setAmount] = useState('');
  const [remark, setRemark] = useState('');
  // 幂等键, 每次打开表单和每次成功后生成新的
  const [key, setKey] = useState(() => nanoid());

  const load = useCallback(async () => {
    try {
      const data = await services.admin.listUserBalances(user.id);
      setBalances(data.balances);
      // 默认选第一个可用货币
      setCurrency((prev) => prev || (data.balances.find((b) => b.is_enabled)?.currency ?? ''));
    } catch (err) {
      toast.error(toErrorMessage(err));
    }
  }, [user.id]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 打开时拉取余额
    void load();
  }, [load]);

  const selected = balances?.find((b) => b.currency === currency);
  const decimals = selected?.decimals ?? 0;
  const parsed = parseAmount(amount, decimals);
  const delta = parsed && parsed > 0 ? (direction === 'add' ? parsed : -parsed) : null;
  const after = selected && delta !== null ? selected.balance + delta : null;

  const handleSubmit = submit(async () => {
    check(selected ? null : '请选择货币');
    check(parsed !== null && parsed > 0 && parsed <= MAX_AMOUNT ? null : '金额格式不正确');
    check(after !== null && after < 0 ? '扣款后余额不能为负' : null);
    check(remark.trim() ? null : '请填写备注');
    await services.admin.adjustUserBalance(user.id, {
      currency,
      amount: delta!,
      remark: remark.trim(),
      idempotency_key: key,
    });
    toast.success(
        `已给 ${user.username} ${direction === 'add' ? '加款' : '扣款'} ${formatAmount(parsed!, decimals)} ${selected!.name}`,
    );
    setAmount('');
    setRemark('');
    setKey(nanoid());
    await load();
  });

  return (
    <div className="flex flex-col gap-4">
      {balances ? (
        <div className="grid grid-cols-2 gap-2">
          {balances.map((b) => (
            <div key={b.currency} className={cn('flex flex-col gap-1 rounded-lg border p-3', !b.is_enabled && 'opacity-60')}>
              <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <CurrencyIcon currency={b.currency} className="size-3.5" />
                {b.name}
                {!b.is_enabled && (
                  <Badge variant="outline" className="px-1 py-0 text-[10px]">
                    已停用
                  </Badge>
                )}
              </span>
              <span className="font-mono text-lg font-semibold tabular-nums">{formatAmount(b.balance, b.decimals)}</span>
              {b.frozen > 0 && (
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  <Snowflake className="size-3" />
                  冻结 {formatAmount(b.frozen, b.decimals)}
                </span>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-2">
          <Skeleton className="h-18 rounded-lg" />
          <Skeleton className="h-18 rounded-lg" />
        </div>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <DirectionToggle value={direction} onChange={setDirection} labels={['加款', '扣款']} disabled={pending} />
        <div className="grid grid-cols-[8rem_1fr] gap-3">
          <div className="flex flex-col gap-2">
            <Label>货币</Label>
            <Select value={currency} onValueChange={setCurrency} disabled={pending}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="选择货币" />
              </SelectTrigger>
              <SelectContent>
                {/* 已停用的货币不能调账 */}
                {balances
                    ?.filter((b) => b.is_enabled)
                    .map((b) => (
                      <SelectItem key={b.currency} value={b.currency}>
                        <CurrencyIcon currency={b.currency} />
                        {b.name}
                      </SelectItem>
                    ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="adjust-amount">金额</Label>
            <Input
              id="adjust-amount"
              inputMode={decimals > 0 ? 'decimal' : 'numeric'}
              placeholder="大于 0"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              disabled={pending}
            />
          </div>
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="adjust-remark">备注</Label>
          <Input
            id="adjust-remark"
            maxLength={255}
            placeholder="调整原因, 用户可以看到"
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            disabled={pending}
          />
        </div>
        <DialogFooter className="items-center">
          <span className={cn('mr-auto text-sm text-muted-foreground', after !== null && after < 0 && 'text-destructive')}>
            {after !== null && `调整后 ${formatAmount(after, decimals)} ${selected!.name}`}
          </span>
          <Button type="button" variant="outline" asChild>
            <Link href={transactionsHref('wallet', user.id)}>
              <ScrollText />
              查看流水
            </Link>
          </Button>
          <SubmitButton pending={pending} variant={direction === 'sub' ? 'destructive' : 'default'}>
            {direction === 'add' ? '加款' : '扣款'}
          </SubmitButton>
        </DialogFooter>
      </form>
    </div>
  );
}
