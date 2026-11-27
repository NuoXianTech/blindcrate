'use client';

import {useCallback, useEffect, useRef, useState} from 'react';
import {X} from 'lucide-react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {QuantityStepper, SubmitButton} from '@/components/common/form';
import {ItemImage} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import {usePlayerSettings} from '@/components/common/player-settings/PlayerSettingsProvider';
import {CurrencyIcon} from '@/components/common/wallet';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {ApiError, toErrorMessage} from '@/lib/services/core';
import type {PlayerSettings} from '@/lib/services/player-settings';
import type {SellOrder} from '@/lib/services/inventory';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatAmount, parseAmount} from '@/lib/utils';
import {MAX_SELL_LINES} from './constants';
import {formatFeeRate, sellableQuantity, sellTotals, type SellLine} from './sell';

/**
 * 出售对话框的Props接口
 */
interface SellDialogProps {
  /** 要出售的物品和默认数量, null 时关闭 */
  lines: SellLine[] | null;
  /** 货币列表, 用于展示货币名称和小数位 */
  currencies: BalanceItem[];
  onClose: () => void;
  /** 出售成功后刷新背包 */
  onSold: (order: SellOrder) => void;
}

/**
 * 出售确认框: 调整每种物品的数量, 按货币汇总卖出金额、手续费和实得
 */
export function SellDialog({lines, ...props}: SellDialogProps) {
  return (
    <Dialog open={!!lines} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>出售物品</DialogTitle>
          <DialogDescription>按物品当前的出售价格成交, 卖出金额扣除手续费后到账</DialogDescription>
        </DialogHeader>
        {/* 每次打开都重新挂载, 重置数量和幂等键 */}
        {lines && <SellForm initial={lines} {...props} />}
      </DialogContent>
    </Dialog>
  );
}

/** 表单中的一行, 数量为输入框的原文 */
interface DraftLine {
  item: SellLine['item'];
  text: string;
}

/** 一行的数量, 格式不对或超出持有数量时为 null */
function quantityOf(line: DraftLine, keepOne: boolean): number | null {
  const max = sellableQuantity(line.item, keepOne);
  if (max === 0) return 0;
  const value = parseAmount(line.text);
  return value !== null && value >= 1 && value <= max ? value : null;
}

/**
 * 出售表单
 */
function SellForm({initial, currencies, onClose, onSold}: Omit<SellDialogProps, 'lines'> & { initial: SellLine[] }) {
  const {qualityOf} = useQualities();
  const {settings, refresh: refreshSettings, openDialog: openSettings} = usePlayerSettings();
  const [snapshot, setSnapshot] = useState<PlayerSettings | null>(null);
  const [feeRate, setFeeRate] = useState(0);
  const [reading, setReading] = useState(true);
  const [readError, setReadError] = useState('');
  const readID = useRef(0);
  const {pending, submit} = useSubmit();
  const [lines, setLines] = useState<DraftLine[]>(() => initial.map((l) => ({item: l.item, text: String(l.quantity)})));
  const drafts = useRef(lines);
  // 幂等键在出售内容变化时重新生成, 内容不变的重复提交(如超时后重试)只成交一次
  const [idempotencyKey, setIdempotencyKey] = useState(() => nanoid());
  const invalidateRead = useCallback(() => {
    readID.current++;
  }, []);

  const readCurrent = useCallback(async () => {
    const id = ++readID.current;
    setReading(true);
    setReadError('');
    try {
      const [preferences, inventory, publicSettings] = await Promise.all([
        refreshSettings(),
        services.inventory.getSellItems(initial.map((l) => l.item.item_id)),
        services.settings.getPublic(),
      ]);
      if (id !== readID.current) return;
      if (!publicSettings.economy) throw new ApiError('无法读取出售手续费, 请重试');
      const byID = new Map(inventory.items.map((it) => [it.item_id, it]));
      const next = drafts.current.map((line) => {
        const item = byID.get(line.item.item_id) ?? {...line.item, quantity: 0};
        const max = sellableQuantity(item, preferences.keep_one_collectible);
        // 只减少超出的数量; 关闭保护或持有数量增加时不擅自增加出售数量。
        const quantity = parseAmount(line.text);
        const text = max === 0 ? '0' : quantity !== null && quantity > max ? String(max) : line.text;
        return {item, text};
      });
      if (next.some((line, i) => line.text !== drafts.current[i].text || line.item.sell_price !== drafts.current[i].item.sell_price || line.item.sell_currency !== drafts.current[i].item.sell_currency)) setIdempotencyKey(nanoid());
      drafts.current = next;
      setLines(next);
      setSnapshot(preferences);
      setFeeRate(publicSettings.economy.sell_fee_rate);
    } catch (err) {
      if (id === readID.current) setReadError(toErrorMessage(err));
    } finally {
      if (id === readID.current) setReading(false);
    }
  }, [initial, refreshSettings]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 打开或恢复确认框时重读设置和库存
    void readCurrent();
    return invalidateRead;
  }, [readCurrent, invalidateRead]);

  const updateLines = (next: DraftLine[]) => {
    drafts.current = next;
    setLines(next);
    setIdempotencyKey(nanoid());
  };
  const setText = (itemID: number, text: string) =>
    updateLines(lines.map((l) => (l.item.item_id === itemID ? {...l, text} : l)));

  const keepOne = snapshot?.keep_one_collectible ?? false;
  const changed = !!snapshot && settings?.version !== snapshot.version;
  const ready = !!snapshot && !reading && !readError && !changed;
  const valid = lines.flatMap((l) => {
    const quantity = quantityOf(l, keepOne);
    return quantity === null || quantity === 0 ? [] : [{item: l.item, quantity}];
  });
  const totals = sellTotals(valid, feeRate);
  const currencyOf = (code: string) => currencies.find((c) => c.currency === code);

  const handleSubmit = (event: React.FormEvent) => submit(async () => {
    check(ready ? null : '请先读取并确认最新的游戏设置和库存');
    check(lines.length === 0 ? '请选择要出售的物品' : null);
    check(lines.length > MAX_SELL_LINES ? `一次最多出售 ${MAX_SELL_LINES} 种物品` : null);
    for (const l of lines) {
      const max = sellableQuantity(l.item, keepOne);
      check(quantityOf(l, keepOne) === null ? `${l.item.name}的数量需为 1-${max}` : null);
    }
    check(valid.length ? null : '没有可出售的物品, 需要保留的藏品已跳过');
    const order = await services.inventory.sell({
      items: valid.map(({item, quantity}) => ({
        item_id: item.item_id,
        quantity,
        sell_currency: item.sell_currency,
        sell_price: item.sell_price,
      })),
      idempotency_key: idempotencyKey,
      fee_rate: feeRate,
      game_settings_version: snapshot!.version,
    }).catch(async (err: unknown) => {
      // 版本、库存或价格冲突后重读, 只更新确认框, 不自动重试出售。
      if (err instanceof ApiError && (err.status === 409 || err.status === 403)) await readCurrent();
      throw err;
    });
    const income = order.incomes
        .map((i) => `${formatAmount(i.amount - i.fee, currencyOf(i.currency)?.decimals ?? 0)} ${currencyOf(i.currency)?.name ?? i.currency}`)
        .join('、');
    toast.success(`已出售, 到账 ${income}`);
    onSold(order);
    onClose();
  })(event);

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex flex-col gap-2 text-sm">
        <span>{reading ? '正在读取设置和库存…' : snapshot ? keepOne ? '收藏保护已开启 · 每种藏品保留 1 件' : '收藏保护已关闭' : '暂时无法确认出售内容'}</span>
        {(readError || changed) && <p role="alert" className="text-destructive">{readError || '游戏设置已变化, 请重新读取并确认出售数量。'}</p>}
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" size="sm" disabled={reading || pending} onClick={() => void readCurrent()}>重新读取</Button>
          <Button
            type="button"
            variant="link"
            size="sm"
            className="px-0 text-muted-foreground"
            onClick={() => {
              // 先关闭出售框再打开设置, 改完设置后重新发起出售
              onClose();
              openSettings();
            }}
          >
            调整游戏设置
          </Button>
        </div>
      </div>
      <ul className="flex flex-col divide-y rounded-lg border">
        {lines.map((l) => {
          const q = qualityOf(l.item.quality);
          const decimals = currencyOf(l.item.sell_currency)?.decimals ?? 0;
          const quantity = quantityOf(l, keepOne);
          const max = sellableQuantity(l.item, keepOne);
          return (
            <li key={l.item.item_id} className="grid grid-cols-[2.5rem_minmax(0,1fr)_2rem] items-start gap-x-3 gap-y-2 p-3">
              <span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-card p-1">
                <ItemImage item={l.item} />
              </span>
              <div className="flex min-w-0 flex-col gap-1">
                <span className={cn('truncate text-sm font-medium', q.text)} title={l.item.name}>
                  {l.item.name}
                </span>
                <span className="text-xs text-muted-foreground">持有 {l.item.quantity} · 可售 {max} · 出售后剩余 {quantity === null ? '-' : l.item.quantity - quantity}</span>
                <span className="flex flex-wrap items-center gap-1 text-xs whitespace-nowrap text-muted-foreground">
                  <CurrencyIcon currency={l.item.sell_currency} className="size-3.5" />
                  <span className="font-mono tabular-nums">{formatAmount(l.item.sell_price, decimals)}</span>
                  <span>× {quantity ?? '-'}</span>
                  {quantity !== null && (
                    <span className="font-mono text-foreground tabular-nums">
                      = {formatAmount(l.item.sell_price * quantity, decimals)}
                    </span>
                  )}
                </span>
              </div>
              <div className="col-span-3 row-start-2 flex justify-end">
                {max === 0 ? <span className="text-xs text-muted-foreground">{l.item.quantity > 0 && keepOne && l.item.category === 'collectible' && !l.item.is_placeholder ? '保留收藏，已跳过' : '无可售数量，已跳过'}</span> : <QuantityStepper
                  label="出售数量"
                  text={l.text}
                  max={max}
                  maxLabel="可售全部"
                  invalid={quantity === null}
                  onChange={(text) => setText(l.item.item_id, text)}
                  disabled={pending || !ready}
                />}
              </div>
              {lines.length > 1 && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="不卖这个"
                  aria-label="不卖这个"
                  className="col-start-3 row-start-1 size-8 hover:text-destructive"
                  onClick={() => updateLines(lines.filter((x) => x.item.item_id !== l.item.item_id))}
                  disabled={pending || reading}
                >
                  <X />
                </Button>
              )}
            </li>
          );
        })}
      </ul>

      <dl className="flex flex-col gap-2 rounded-lg border bg-card p-4 text-sm">
        {totals.length === 0 && <span className="text-muted-foreground">请填写出售数量</span>}
        {totals.map((t) => {
          const decimals = currencyOf(t.currency)?.decimals ?? 0;
          return (
            <div key={t.currency} className="flex flex-col gap-1">
              <SummaryRow label="卖出金额" currency={t.currency} value={formatAmount(t.amount, decimals)} />
              <SummaryRow label={`手续费 ${formatFeeRate(feeRate)}`} currency={t.currency} value={formatAmount(-t.fee, decimals, {signed: true})} muted />
              <SummaryRow label="实得" currency={t.currency} value={formatAmount(t.amount - t.fee, decimals)} strong />
            </div>
          );
        })}
      </dl>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending} disabled={!ready || !valid.length}>确认出售</SubmitButton>
      </DialogFooter>
    </form>
  );
}

/**
 * 汇总中的一行
 */
function SummaryRow({label, currency, value, muted, strong}: {
  label: string;
  currency: string;
  value: string;
  muted?: boolean;
  strong?: boolean;
}) {
  return (
    <div className={cn('flex items-center justify-between gap-4', muted && 'text-muted-foreground', strong && 'font-bold')}>
      <dt>{label}</dt>
      <dd className="flex items-center gap-1.5">
        <CurrencyIcon currency={currency} className="size-4" />
        <span className="font-mono tabular-nums">{value}</span>
      </dd>
    </div>
  );
}
