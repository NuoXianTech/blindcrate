'use client';

import {useEffect, useState} from 'react';
import {Coins, Package, Trash2} from 'lucide-react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Skeleton} from '@/components/ui/skeleton';
import {SubmitButton} from '@/components/common/form';
import {ItemCombobox} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import {useAdminCatalog} from '@/hooks/use-admin-catalog';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminItem, Currency, LootEntry, LootEntryRequest, LootPoolListItem} from '@/lib/services/admin';
import {toErrorMessage} from '@/lib/services/core';
import {amountInput, parseAmount} from '@/lib/utils';
import {MAX_CURRENCY_AMOUNT, MAX_ENTRIES, MAX_ITEM_QUANTITY, MAX_WEIGHT} from './constants';

/** 编辑中的一个条目, 数值保留输入框里的原文 */
interface Draft {
  key: string;
  /** 物品条目的物品 ID, 货币条目为 null */
  itemID: number | null;
  /** 货币条目的货币代码, 物品条目为空 */
  currency: string;
  weight: string;
  min: string;
  max: string;
}

/**
 * 编辑掉落池的条目: 物品或货币、权重和每次掉落的数量范围, 保存时整体替换
 */
export function EntriesDialog({pool, onClose, onSaved}: {
  pool: LootPoolListItem | null;
  onClose: () => void;
  /** 保存成功后刷新列表中的条目数 */
  onSaved: () => void;
}) {
  return (
    <Dialog open={!!pool} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{pool?.name} 的条目</DialogTitle>
          <DialogDescription>
            每次抽取按权重选中一个条目, 概率 = 权重 / 可抽取条目的权重之和; 已下架的物品和已停用的货币不参与抽取
          </DialogDescription>
        </DialogHeader>
        {/* key 保证切换掉落池时重新拉取 */}
        {pool && <EntriesEditor key={pool.id} pool={pool} onClose={onClose} onSaved={onSaved} />}
      </DialogContent>
    </Dialog>
  );
}

function toDraft(entry: LootEntry, currencies: Currency[]): Draft {
  const decimals = entry.currency ? currencies.find((c) => c.code === entry.currency)?.decimals ?? 0 : 0;
  return {
    key: String(entry.id),
    itemID: entry.item_id || null,
    currency: entry.currency,
    weight: String(entry.weight),
    min: amountInput(entry.min_quantity, decimals),
    max: amountInput(entry.max_quantity, decimals),
  };
}

/**
 * 条目列表和保存按钮
 */
function EntriesEditor({pool, onClose, onSaved}: { pool: LootPoolListItem; onClose: () => void; onSaved: () => void }) {
  const {pending, submit} = useSubmit();
  const {items, currencies, loaded} = useAdminCatalog();
  const [entries, setEntries] = useState<LootEntry[] | null>(null);
  const [drafts, setDrafts] = useState<Draft[]>([]);

  useEffect(() => {
    services.admin.listLootPoolEntries(pool.id)
        .then((data) => setEntries(data.entries))
        .catch((err) => toast.error(toErrorMessage(err)));
  }, [pool.id]);

  // 条目和货币都到了才能换算货币金额的小数位
  const ready = loaded && entries !== null;
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 数据到齐后生成可编辑的条目
    if (ready) setDrafts(entries.map((e) => toDraft(e, currencies)));
  }, [ready, entries, currencies]);

  const itemMap = new Map(items.map((i) => [i.id, i]));
  const decimalsOf = (d: Draft) => (d.currency ? currencies.find((c) => c.code === d.currency)?.decimals ?? 0 : 0);
  const available = (d: Draft) =>
    d.itemID !== null ? !!itemMap.get(d.itemID)?.is_enabled : !!currencies.find((c) => c.code === d.currency)?.is_enabled;
  const weightOf = (d: Draft) => {
    const weight = parseAmount(d.weight);
    return weight && weight > 0 && available(d) ? weight : 0;
  };
  const totalWeight = drafts.reduce((sum, d) => sum + weightOf(d), 0);

  const update = (key: string, patch: Partial<Draft>) =>
    setDrafts((prev) => prev.map((d) => (d.key === key ? {...d, ...patch} : d)));
  const add = (kind: 'item' | 'currency') =>
    setDrafts((prev) => [
      ...prev,
      {key: nanoid(), itemID: null, currency: kind === 'currency' ? currencies[0]?.code ?? '' : '', weight: '1', min: '1', max: '1'},
    ]);

  const handleSubmit = submit(async () => {
    check(drafts.length > MAX_ENTRIES ? `最多 ${MAX_ENTRIES} 个条目` : null);
    const body = drafts.map((d, i): LootEntryRequest => {
      const row = `第 ${i + 1} 行`;
      const isItem = !d.currency;
      check(isItem && d.itemID === null ? `${row}: 请选择物品` : null);
      const weight = parseAmount(d.weight) ?? 0;
      check(weight < 1 || weight > MAX_WEIGHT ? `${row}: 权重需为 1-${MAX_WEIGHT} 的整数` : null);
      const decimals = decimalsOf(d);
      const min = parseAmount(d.min, decimals) ?? 0;
      const max = parseAmount(d.max, decimals) ?? 0;
      const limit = isItem ? MAX_ITEM_QUANTITY : MAX_CURRENCY_AMOUNT;
      check(min < 1 || max < 1 || max > limit ? `${row}: 数量格式不正确` : null);
      check(min > max ? `${row}: 最小数量不能大于最大数量` : null);
      return {item_id: d.itemID ?? 0, currency: d.currency, weight, min_quantity: min, max_quantity: max};
    });
    const data = await services.admin.replaceLootPoolEntries(pool.id, body);
    setEntries(data.entries);
    toast.success(`已保存 ${pool.name} 的 ${data.entries.length} 个条目`);
    onSaved();
  });

  if (!ready) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-9" />
        <Skeleton className="h-9" />
        <Skeleton className="h-9" />
      </div>
    );
  }

  // 已选过的物品不能再选, 同一物品只需一个条目
  const chosen = new Set(drafts.map((d) => d.itemID));
  const optionsFor = (d: Draft) =>
    items
        .filter((i) => i.id === d.itemID || (i.is_enabled && !chosen.has(i.id)))
        .map((i) => ({id: i.id, name: i.name, quality: i.quality}));

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex max-h-[60vh] flex-col gap-2 overflow-y-auto pr-1">
        {drafts.length > 0 && (
          <div className="grid grid-cols-[1fr_5.5rem_11rem_4rem_2.25rem] gap-2 px-1 text-xs text-muted-foreground">
            <span>掉落</span>
            <span>权重</span>
            <span>每次数量</span>
            <span className="text-right">概率</span>
            <span />
          </div>
        )}
        {drafts.map((d) => (
          <EntryRow
            key={d.key}
            draft={d}
            item={d.itemID !== null ? itemMap.get(d.itemID) : undefined}
            currencies={currencies}
            options={optionsFor(d)}
            available={available(d)}
            probability={totalWeight > 0 ? weightOf(d) / totalWeight : 0}
            disabled={pending}
            onChange={(patch) => update(d.key, patch)}
            onRemove={() => setDrafts((prev) => prev.filter((x) => x.key !== d.key))}
          />
        ))}
        {drafts.length === 0 && <p className="py-6 text-center text-sm text-muted-foreground">还没有条目, 抽取时会失败</p>}
      </div>

      <div className="flex gap-2">
        <Button type="button" variant="outline" size="sm" onClick={() => add('item')} disabled={pending}>
          <Package />
          添加物品
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={() => add('currency')} disabled={pending || !currencies.length}>
          <Coins />
          添加货币
        </Button>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          关闭
        </Button>
        <SubmitButton pending={pending}>保存条目</SubmitButton>
      </DialogFooter>
    </form>
  );
}

/**
 * 条目行的Props接口
 */
interface EntryRowProps {
  draft: Draft;
  /** 物品条目对应的物品, 已删除时为空 */
  item?: AdminItem;
  currencies: Currency[];
  options: { id: number; name: string; quality: number }[];
  /** 是否参与抽取 */
  available: boolean;
  /** 按当前输入估算的每次抽中概率 */
  probability: number;
  disabled: boolean;
  onChange: (patch: Partial<Draft>) => void;
  onRemove: () => void;
}

/**
 * 一个条目: 物品下拉框或货币选择, 权重, 数量范围和估算概率
 */
function EntryRow({draft, item, currencies, options, available, probability, disabled, onChange, onRemove}: EntryRowProps) {
  const isItem = !draft.currency;
  const decimals = isItem ? 0 : currencies.find((c) => c.code === draft.currency)?.decimals ?? 0;
  const inputMode = decimals > 0 ? 'decimal' : 'numeric';
  return (
    <div className={cn('grid grid-cols-[1fr_5.5rem_11rem_4rem_2.25rem] items-center gap-2', !available && 'opacity-60')}>
      <div className="flex min-w-0 items-center gap-1.5">
        {isItem ? (
          <ItemCombobox
            options={options}
            value={draft.itemID}
            onChange={(id) => onChange({itemID: id})}
            placeholder={draft.itemID !== null && !item ? `已删除的物品 #${draft.itemID}` : '选择物品'}
            disabled={disabled}
          />
        ) : (
          <Select value={draft.currency} onValueChange={(currency) => onChange({currency})} disabled={disabled}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {currencies.map((c) => (
                <SelectItem key={c.code} value={c.code}>
                  <CurrencyIcon currency={c.code} />
                  {c.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        {!available && (draft.itemID !== null || !isItem) && (
          <Badge variant="outline" className="shrink-0" title="不参与抽取">
            {isItem ? '已下架' : '已停用'}
          </Badge>
        )}
      </div>
      <Input
        inputMode="numeric"
        aria-label="权重"
        className="font-mono tabular-nums"
        value={draft.weight}
        onChange={(e) => onChange({weight: e.target.value})}
        disabled={disabled}
      />
      <div className="flex items-center gap-1">
        <Input
          inputMode={inputMode}
          aria-label="最小数量"
          className="font-mono tabular-nums"
          value={draft.min}
          onChange={(e) => onChange({min: e.target.value})}
          disabled={disabled}
        />
        <span className="text-muted-foreground">-</span>
        <Input
          inputMode={inputMode}
          aria-label="最大数量"
          className="font-mono tabular-nums"
          value={draft.max}
          onChange={(e) => onChange({max: e.target.value})}
          disabled={disabled}
        />
      </div>
      <span className="text-right font-mono text-sm tabular-nums">{(probability * 100).toFixed(1)}%</span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        title="删除条目"
        aria-label="删除条目"
        className="hover:text-destructive"
        onClick={onRemove}
        disabled={disabled}
      >
        <Trash2 />
      </Button>
    </div>
  );
}
