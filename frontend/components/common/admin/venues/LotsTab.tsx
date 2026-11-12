'use client';

import {ChevronRight, Plus, Trash2} from 'lucide-react';
import {nanoid} from 'nanoid';
import {Button} from '@/components/ui/button';
import type {LootPoolListItem} from '@/lib/services/loot';
import type {LotTier, TierStats} from '@/lib/services/venue';
import type {Currency} from '@/lib/services/wallet';
import {formatAmount} from '@/lib/utils';
import {FieldGroup, InputField} from './Fields';
import {WarehouseEditor} from './WarehouseEditor';
import {MAX_GAP_SECONDS, MAX_LOTS, MAX_TIERS} from './constants';
import type {FormTabProps, TierDraft} from './form';

/**
 * 拍场: 一局开拍几个拍品、拍品间隔、带入覆盖率和拍品的档次; 最多拍品数为 1 时一局一个仓库
 */
export function LotsTab({form, onChange, disabled, pools, currency, savedTiers, tierStats, stale}: FormTabProps & {
  pools: LootPoolListItem[];
  /** 出价货币, 金额按它的小数位展示 */
  currency?: Currency;
  /** 上次保存时的档次, 带推导出的金额; 新建时为空 */
  savedTiers: LotTier[];
  /** 最近一次模拟的各档次价值分布 */
  tierStats: TierStats[];
  /** 推导之后表单有改动 */
  stale: boolean;
}) {
  // 一局一个仓库, 带入资金按单个仓库的价值推导, 拍品间隔和带入覆盖率不生效
  const single = form.lotMax.trim() === '1';
  const updateTier = (key: string, patch: Partial<TierDraft>) =>
    onChange({tiers: form.tiers.map((t) => (t.key === key ? {...t, ...patch} : t))});
  const addTier = () => {
    const n = form.tiers.length + 1;
    // 复制最后一种档次的仓库
    const last = form.tiers.at(-1)?.warehouse;
    if (!last) return;
    const warehouse = {
      ...last,
      slots: last.slots.map((s) => ({...s, key: nanoid()})),
      swings: last.swings.map((s) => ({...s, key: nanoid()})),
    };
    onChange({tiers: [...form.tiers, {key: nanoid(), code: `tier${n}`, name: `档次 ${n}`, weight: '1', warehouse}]});
  };

  return (
    <div className="flex flex-col gap-6">
      <FieldGroup
        title="拍品数"
        help={
          single ?
            '一局一个仓库, 带入资金上限按单个仓库的价值分布推导; 最多拍品数大于 1 时一局依次开拍多个拍品, 共用一笔带入资金' :
            '一局依次开拍若干个拍品, 共用一笔带入资金; 拍品数和每个拍品的格子大小开局时公开, 档次拍场结束后公开'
        }
      >
        <div className="grid gap-3 sm:grid-cols-2">
          <InputField
            id="venue-lot-min"
            label="最少拍品数"
            help={`1-${MAX_LOTS} 个`}
            inputMode="numeric"
            value={form.lotMin}
            onChange={(lotMin) => onChange({lotMin})}
            disabled={disabled}
          />
          <InputField
            id="venue-lot-max"
            label="最多拍品数"
            help="每局在最少和最多之间均匀随机; 1 表示一局一个仓库"
            inputMode="numeric"
            value={form.lotMax}
            onChange={(lotMax) => onChange({lotMax})}
            disabled={disabled}
          />
          <InputField
            id="venue-gap-seconds"
            label="拍品间隔"
            help={`一个拍品开箱后停留多久进入下一个, 0-${MAX_GAP_SECONDS} 秒`}
            suffix="秒"
            inputMode="numeric"
            value={form.gapSeconds}
            onChange={(gapSeconds) => onChange({gapSeconds})}
            disabled={disabled || single}
          />
          <InputField
            id="venue-coverage"
            label="带入覆盖率"
            help="带入资金上限 = 拍品数的期望 × 单个拍品的期望价值 × 覆盖率, 最高 200%; 0 表示用玩法平衡设置的默认值"
            suffix="%"
            inputMode="decimal"
            value={form.coverage}
            onChange={(coverage) => onChange({coverage})}
            disabled={disabled || single}
          />
        </div>
      </FieldGroup>

      <FieldGroup
        title="档次"
        help={`每个拍品按权重抽一种档次生成仓库; 格子大小相同的档次起拍价等金额相同; 1-${MAX_TIERS} 种`}
      >
        <div className="flex flex-col gap-3">
          {form.tiers.map((t, i) => (
            <TierCard
              key={t.key}
              index={i}
              tier={t}
              pools={pools}
              saved={savedTiers.find((x) => x.code === t.code.trim())}
              stats={tierStats.find((x) => x.code === t.code.trim())}
              currency={currency}
              stale={stale}
              onChange={(patch) => updateTier(t.key, patch)}
              onRemove={() => onChange({tiers: form.tiers.filter((x) => x.key !== t.key)})}
              removable={form.tiers.length > 1}
              disabled={disabled}
            />
          ))}
          <div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={addTier}
              disabled={disabled || form.tiers.length >= MAX_TIERS}
            >
              <Plus />
              添加档次
            </Button>
          </div>
        </div>
      </FieldGroup>
    </div>
  );
}

/**
 * 档次卡片的Props接口
 */
interface TierCardProps {
  index: number;
  tier: TierDraft;
  pools: LootPoolListItem[];
  /** 上次保存时的同代码档次 */
  saved?: LotTier;
  /** 最近一次模拟的同代码档次 */
  stats?: TierStats;
  currency?: Currency;
  stale: boolean;
  onChange: (patch: Partial<TierDraft>) => void;
  onRemove: () => void;
  /** 至少保留一种档次 */
  removable: boolean;
  disabled: boolean;
}

/**
 * 一种档次: 代码、名称、权重和仓库生成规则, 以及推导出的金额(只读)
 */
function TierCard({index, tier, pools, saved, stats, currency, stale, onChange, onRemove, removable, disabled}: TierCardProps) {
  const id = `venue-tier-${tier.key}`;
  const amount = (value: number) => formatAmount(value, currency?.decimals ?? 0);
  const derived = saved && saved.auction.start_price > 0 ? saved.auction : null;
  const slots = tier.warehouse.slots.length;

  return (
    <div className="flex flex-col gap-3 rounded-lg border p-3">
      <div className="flex items-start gap-2">
        <div className="grid flex-1 gap-3 sm:grid-cols-[1fr_1fr_6rem]">
          <InputField
            id={`${id}-code`}
            label={`第 ${index + 1} 种 · 代码`}
            help="档次中唯一"
            maxLength={32}
            value={tier.code}
            onChange={(code) => onChange({code})}
            disabled={disabled}
          />
          <InputField
            id={`${id}-name`}
            label="名称"
            help="管理端和拍场结束后显示"
            maxLength={32}
            value={tier.name}
            onChange={(name) => onChange({name})}
            disabled={disabled}
          />
          <InputField
            id={`${id}-weight`}
            label="权重"
            help="1-10000"
            inputMode="numeric"
            value={tier.weight}
            onChange={(weight) => onChange({weight})}
            disabled={disabled}
          />
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          title="删除档次"
          aria-label="删除档次"
          className="mt-6 hover:text-destructive"
          onClick={onRemove}
          disabled={disabled || !removable}
        >
          <Trash2 />
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-x-3 gap-y-0.5 rounded-md bg-muted/50 px-3 py-2 text-xs sm:grid-cols-3">
        <span className="text-muted-foreground">
          期望价值
          <span className="ml-1.5 font-mono text-foreground tabular-nums">
            {stats ? amount(stats.ranked.mean) : derived ? amount(derived.mean) : '-'}
          </span>
        </span>
        <span className="text-muted-foreground">
          起拍价
          <span className="ml-1.5 font-mono text-foreground tabular-nums">{derived ? amount(derived.start_price) : '-'}</span>
        </span>
        <span className="text-muted-foreground">
          速胜下限
          <span className="ml-1.5 font-mono text-foreground tabular-nums">{derived ? amount(derived.speed_floor) : '-'}</span>
        </span>
        <span className="col-span-full text-muted-foreground">
          {derived ?
            `期望价值取${stats ? '最近一次模拟' : '上次保存'}的结果, 金额为上次保存时推导的${stale ? ', 配置已修改' : ''}; 保存后更新` :
            '金额保存时推导; 期望价值模拟后显示'}
        </span>
      </div>

      <details className="group">
        <summary className="flex cursor-pointer list-none items-center gap-1 text-sm text-muted-foreground select-none">
          <ChevronRight className="size-4 transition-transform group-open:rotate-90" />
          仓库生成规则
          <span className="text-xs">
            · {tier.warehouse.gridWidth}×{tier.warehouse.gridHeight} 格 · {slots} 个档位
          </span>
        </summary>
        <div className="mt-3 flex flex-col gap-6">
          <WarehouseEditor
            id={id}
            value={tier.warehouse}
            onChange={(patch) => onChange({warehouse: {...tier.warehouse, ...patch}})}
            disabled={disabled}
            pools={pools}
          />
        </div>
      </details>
    </div>
  );
}
