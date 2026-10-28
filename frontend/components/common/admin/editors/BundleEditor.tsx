'use client';

import {Coins, Package, Trash2} from 'lucide-react';
import {nanoid} from 'nanoid';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {ItemCombobox} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import type {AdminItem, Currency} from '@/lib/services/admin';
import {decimalsOf, type BundleDraft} from './drafts';

/**
 * 货币和物品编辑器的Props接口
 */
interface BundleEditorProps {
  drafts: BundleDraft[];
  onChange: (drafts: BundleDraft[]) => void;
  /** 含已下架的物品, 用于显示已选物品的名称 */
  items: AdminItem[];
  currencies: Currency[];
  /** 最多的行数, 达到后不能再添加 */
  max: number;
  /** 没有任何一行时的提示 */
  empty: string;
  disabled?: boolean;
}

/**
 * 编辑一组货币和物品, 如入场代价; 每行一种货币或物品
 */
export function BundleEditor({drafts, onChange, items, currencies, max, empty, disabled}: BundleEditorProps) {
  const itemMap = new Map(items.map((i) => [i.id, i]));
  // 已选过的物品不能再选, 同一物品只需一行
  const chosen = new Set(drafts.map((d) => d.itemID));
  const update = (key: string, patch: Partial<BundleDraft>) =>
    onChange(drafts.map((d) => (d.key === key ? {...d, ...patch} : d)));
  const add = (kind: 'item' | 'currency') =>
    onChange([...drafts, {key: nanoid(), itemID: null, currency: kind === 'currency' ? currencies[0]?.code ?? '' : '', amount: '1'}]);
  const full = drafts.length >= max;

  return (
    <div className="flex flex-col gap-2">
      {drafts.map((d) => {
        const item = d.itemID !== null ? itemMap.get(d.itemID) : undefined;
        const decimals = d.currency ? decimalsOf(currencies, d.currency) : 0;
        const options = items
            .filter((i) => i.id === d.itemID || (i.is_enabled && !chosen.has(i.id)))
            .map((i) => ({id: i.id, name: i.name, quality: i.quality}));
        return (
          <div key={d.key} className="grid grid-cols-[1fr_9rem_2.25rem] items-center gap-2">
            <div className="flex min-w-0 items-center gap-1.5">
              {d.currency ? (
                <Select value={d.currency} onValueChange={(currency) => update(d.key, {currency})} disabled={disabled}>
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
              ) : (
                <ItemCombobox
                  options={options}
                  value={d.itemID}
                  onChange={(id) => update(d.key, {itemID: id})}
                  placeholder={d.itemID !== null && !item ? `已删除的物品 #${d.itemID}` : '选择物品'}
                  disabled={disabled}
                />
              )}
              {item && !item.is_enabled && (
                <Badge variant="outline" className="shrink-0">
                  已下架
                </Badge>
              )}
            </div>
            <Input
              inputMode={decimals > 0 ? 'decimal' : 'numeric'}
              aria-label={d.currency ? '金额' : '数量'}
              placeholder={d.currency ? '金额' : '数量'}
              className="font-mono tabular-nums"
              value={d.amount}
              onChange={(e) => update(d.key, {amount: e.target.value})}
              disabled={disabled}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              title="删除"
              aria-label="删除"
              className="hover:text-destructive"
              onClick={() => onChange(drafts.filter((x) => x.key !== d.key))}
              disabled={disabled}
            >
              <Trash2 />
            </Button>
          </div>
        );
      })}
      {drafts.length === 0 && <p className="py-2 text-sm text-muted-foreground">{empty}</p>}

      <div className="flex gap-2">
        <Button type="button" variant="outline" size="sm" onClick={() => add('currency')} disabled={disabled || full || !currencies.length}>
          <Coins />
          添加货币
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={() => add('item')} disabled={disabled || full}>
          <Package />
          添加物品
        </Button>
      </div>
    </div>
  );
}
