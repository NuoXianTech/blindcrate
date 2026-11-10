'use client';

import {useMemo, useState} from 'react';
import {Eraser, X} from 'lucide-react';
import {toast} from 'sonner';
import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {ItemCombobox, ItemImage} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {AdminItem} from '@/lib/services/item';
import type {LootPoolListItem} from '@/lib/services/loot';
import {memberStatus} from './constants';

/**
 * 收集组藏品编辑器的Props接口
 */
interface SetItemsEditorProps {
  /** 已选的物品 ID */
  value: number[];
  onChange: (value: number[]) => void;
  /** 含已下架的物品, 用于显示已选物品的名称和状态 */
  items: AdminItem[];
  /** 可以从中批量添加藏品的掉落池 */
  pools: LootPoolListItem[];
  /** 最多的种数, 达到后不能再添加 */
  max: number;
  disabled?: boolean;
}

/**
 * 编辑收集组中的藏品: 逐个添加, 或从掉落池、按品质批量添加上架中的藏品
 * 已下架的标出不计入进度; 已删除或不是藏品的标红, 原来就在组内的可以保留, 同样不计入
 */
export function SetItemsEditor({value, onChange, items, pools, max, disabled}: SetItemsEditorProps) {
  const {qualities, qualityOf} = useQualities();
  const [loadingPool, setLoadingPool] = useState(false);
  // 目录可能有上千种物品, 只在目录或已选的变化时重新计算
  const itemMap = useMemo(() => new Map(items.map((i) => [i.id, i])), [items]);
  const enabled = useMemo(() => items.filter((i) => memberStatus(i) === 'counted'), [items]);
  // 逐个添加时已下架的藏品也可以选, 重新上架后计入进度
  const options = useMemo(() => {
    const chosen = new Set(value);
    return items
        .filter((i) => memberStatus(i) !== 'invalid' && !chosen.has(i.id))
        .map((i) => ({id: i.id, name: i.name, quality: i.quality}));
  }, [items, value]);
  // 按品质从高到低、ID 排列, 与玩家看到的图鉴一致; 找不到的物品排在最后
  const sorted = useMemo(() => [...value].sort((a, b) => {
    const x = itemMap.get(a);
    const y = itemMap.get(b);
    if (!x || !y) return (x ? 0 : 1) - (y ? 0 : 1) || a - b;
    return y.quality - x.quality || a - b;
  }), [value, itemMap]);

  /** 添加物品, 跳过已选的, 超出上限的不添加; 返回添加和因上限跳过的种数 */
  const add = (ids: number[]) => {
    const next = [...value];
    const seen = new Set(value);
    let skipped = 0;
    for (const id of ids) {
      if (seen.has(id)) continue;
      if (next.length >= max) {
        skipped++;
        continue;
      }
      seen.add(id);
      next.push(id);
    }
    onChange(next);
    return {added: next.length - value.length, skipped};
  };

  /** 批量添加后提示结果 */
  const addMany = (ids: number[]) => {
    const {added, skipped} = add(ids);
    if (!added && !skipped) {
      toast.info('没有新的藏品可以添加');
      return;
    }
    toast.success(`添加了 ${added} 种藏品${skipped ? `, 超出上限的 ${skipped} 种没有添加` : ''}`);
  };

  const addPool = async (poolID: string) => {
    setLoadingPool(true);
    try {
      const data = await services.lootAdmin.listLootPoolEntries(Number(poolID));
      const ids = new Set(data.entries.map((e) => e.item_id).filter((id) => memberStatus(itemMap.get(id)) === 'counted'));
      addMany([...ids]);
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setLoadingPool(false);
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="grid gap-2 sm:grid-cols-3">
        <ItemCombobox
          options={options}
          value={null}
          onChange={(id) => add([id])}
          placeholder="添加藏品"
          disabled={disabled || value.length >= max}
        />
        {/* value 固定为空, 选择后回到占位文字, 可以再次选择同一项 */}
        <Select value="" onValueChange={(id) => void addPool(id)} disabled={disabled || loadingPool || !pools.length}>
          <SelectTrigger className="w-full" aria-label="从掉落池添加">
            <SelectValue placeholder={pools.length ? '从掉落池添加' : '没有掉落池'} />
          </SelectTrigger>
          <SelectContent>
            {pools.map((p) => (
              <SelectItem key={p.id} value={String(p.id)}>
                {p.name}
                <span className="font-mono text-xs text-muted-foreground">{p.code}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value="" onValueChange={(level) => addMany(enabled.filter((i) => i.quality === Number(level)).map((i) => i.id))} disabled={disabled}>
          <SelectTrigger className="w-full" aria-label="按品质添加">
            <SelectValue placeholder="按品质添加" />
          </SelectTrigger>
          <SelectContent>
            {qualities.map((q) => (
              <SelectItem key={q.level} value={String(q.level)}>
                <span className={q.text}>{q.name}</span>
                <span className="text-xs text-muted-foreground tabular-nums">{enabled.filter((i) => i.quality === q.level).length} 种</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex items-center justify-between gap-2 text-sm">
        <span className="text-muted-foreground tabular-nums">
          已选 {value.length}/{max} 种
        </span>
        <Button type="button" variant="ghost" size="sm" onClick={() => onChange([])} disabled={disabled || !value.length}>
          <Eraser />
          清空
        </Button>
      </div>

      {sorted.length ? (
        <div className="flex max-h-72 flex-wrap gap-1.5 overflow-y-auto">
          {sorted.map((id) => {
            const item = itemMap.get(id);
            const status = memberStatus(item);
            const name = item?.name ?? `#${id}`;
            return (
              <span
                key={id}
                className={cn(
                    'flex items-center gap-1.5 rounded-md border bg-background py-0.5 pr-0.5 pl-1',
                    status === 'invalid' && 'border-destructive',
                )}
              >
                <span className="flex size-6 shrink-0 items-center justify-center overflow-hidden rounded">
                  <ItemImage item={{name, image_url: item?.image_url ?? '', is_placeholder: !item}} />
                </span>
                <span className={cn('max-w-32 truncate text-sm', item ? qualityOf(item.quality).text : 'text-muted-foreground')} title={name}>
                  {name}
                </span>
                {status === 'disabled' && <Badge variant="outline">已下架</Badge>}
                {status === 'invalid' && <Badge variant="destructive">{item ? '不是藏品' : '已删除'}</Badge>}
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-6"
                  title="移除"
                  aria-label={`移除 ${name}`}
                  onClick={() => onChange(value.filter((v) => v !== id))}
                  disabled={disabled}
                >
                  <X />
                </Button>
              </span>
            );
          })}
        </div>
      ) : (
        <div className="rounded-lg border border-dashed p-4 text-center text-sm text-muted-foreground">还没有藏品</div>
      )}
    </div>
  );
}
