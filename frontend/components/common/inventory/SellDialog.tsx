'use client';

import {useState} from 'react';
import {Minus, Plus, X} from 'lucide-react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {SubmitButton} from '@/components/common/form';
import {ItemImage} from '@/components/common/item';
import {useQualities, useSite} from '@/components/common/site';
import {CurrencyIcon} from '@/components/common/wallet';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {SellOrder} from '@/lib/services/inventory';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatAmount, parseAmount} from '@/lib/utils';
import {MAX_SELL_LINES, MAX_SELL_QUANTITY} from './constants';
import {formatFeeRate, sellTotals, type SellLine} from './sell';

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
function quantityOf(line: DraftLine): number | null {
  const value = parseAmount(line.text);
  return value !== null && value >= 1 && value <= Math.min(line.item.quantity, MAX_SELL_QUANTITY) ? value : null;
}

/**
 * 出售表单
 */
function SellForm({initial, currencies, onClose, onSold}: Omit<SellDialogProps, 'lines'> & { initial: SellLine[] }) {
  const {economy} = useSite();
  const {qualityOf} = useQualities();
  const feeRate = economy.sell_fee_rate;
  const {pending, submit} = useSubmit();
  const [lines, setLines] = useState<DraftLine[]>(() => initial.map((l) => ({item: l.item, text: String(l.quantity)})));
  // 幂等键在出售内容变化时重新生成, 内容不变的重复提交(如超时后重试)只成交一次
  const [idempotencyKey, setIdempotencyKey] = useState(() => nanoid());

  const updateLines = (next: DraftLine[]) => {
    setLines(next);
    setIdempotencyKey(nanoid());
  };
  const setText = (itemID: number, text: string) =>
    updateLines(lines.map((l) => (l.item.item_id === itemID ? {...l, text} : l)));

  const valid = lines.flatMap((l) => {
    const quantity = quantityOf(l);
    return quantity === null ? [] : [{item: l.item, quantity}];
  });
  const totals = sellTotals(valid, feeRate);
  const currencyOf = (code: string) => currencies.find((c) => c.currency === code);

  const handleSubmit = submit(async () => {
    check(lines.length === 0 ? '请选择要出售的物品' : null);
    check(lines.length > MAX_SELL_LINES ? `一次最多出售 ${MAX_SELL_LINES} 种物品` : null);
    for (const l of lines) {
      const max = Math.min(l.item.quantity, MAX_SELL_QUANTITY);
      check(quantityOf(l) === null ? `${l.item.name}的数量需为 1-${max}` : null);
    }
    const order = await services.inventory.sell({
      items: valid.map(({item, quantity}) => ({
        item_id: item.item_id,
        quantity,
        sell_currency: item.sell_currency,
        sell_price: item.sell_price,
      })),
      idempotency_key: idempotencyKey,
      fee_rate: feeRate,
    });
    const income = order.incomes
        .map((i) => `${formatAmount(i.amount - i.fee, currencyOf(i.currency)?.decimals ?? 0)} ${currencyOf(i.currency)?.name ?? i.currency}`)
        .join('、');
    toast.success(`已出售, 到账 ${income}`);
    onSold(order);
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <ul className="flex flex-col divide-y rounded-lg border">
        {lines.map((l) => {
          const q = qualityOf(l.item.quality);
          const decimals = currencyOf(l.item.sell_currency)?.decimals ?? 0;
          const quantity = quantityOf(l);
          return (
            <li key={l.item.item_id} className="flex items-center gap-3 p-3">
              <span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-card p-1">
                <ItemImage item={l.item} />
              </span>
              <div className="flex min-w-0 flex-1 flex-col gap-1">
                <span className={cn('truncate text-sm font-medium', q.text)} title={l.item.name}>
                  {l.item.name}
                </span>
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
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
              <QuantityStepper
                text={l.text}
                max={Math.min(l.item.quantity, MAX_SELL_QUANTITY)}
                invalid={quantity === null}
                onChange={(text) => setText(l.item.item_id, text)}
                disabled={pending}
              />
              {lines.length > 1 && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title="不卖这个"
                  aria-label="不卖这个"
                  className="size-8 hover:text-destructive"
                  onClick={() => updateLines(lines.filter((x) => x.item.item_id !== l.item.item_id))}
                  disabled={pending}
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
        <SubmitButton pending={pending}>确认出售</SubmitButton>
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

/**
 * 数量步进器: 减、输入框、加, 以及一键填入全部
 */
function QuantityStepper({text, max, invalid, onChange, disabled}: {
  text: string;
  max: number;
  invalid: boolean;
  onChange: (text: string) => void;
  disabled?: boolean;
}) {
  const value = parseAmount(text) ?? 0;
  return (
    <div className="flex shrink-0 items-center gap-1">
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-8"
        aria-label="减少"
        onClick={() => onChange(String(Math.min(max, Math.max(1, value - 1))))}
        disabled={disabled || value <= 1}
      >
        <Minus />
      </Button>
      <Input
        inputMode="numeric"
        aria-label="出售数量"
        aria-invalid={invalid}
        className="h-8 w-16 text-center font-mono tabular-nums"
        value={text}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      />
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-8"
        aria-label="增加"
        onClick={() => onChange(String(Math.min(max, Math.max(1, value + 1))))}
        disabled={disabled || value >= max}
      >
        <Plus />
      </Button>
      <Button type="button" variant="ghost" size="sm" className="h-8 px-2" onClick={() => onChange(String(max))} disabled={disabled || value === max}>
        全部
      </Button>
    </div>
  );
}
