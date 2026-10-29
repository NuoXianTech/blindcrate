'use client';

import {Plus, Trash2} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {BundleEditor} from '@/components/common/admin/editors';
import {TierName} from '@/components/common/ranked';
import type {AdminItem, Currency} from '@/lib/services/admin';
import type {Tier} from '@/lib/services/ranked';
import {tierOf} from '@/lib/ranked';
import {parseAmount} from '@/lib/utils';
import {MAX_REWARD_LINES, MAX_REWARDS} from './constants';
import {newRewardDraft, type RewardDraft} from './rewards';

/**
 * 赛季奖励编辑器的Props接口
 */
interface RewardsEditorProps {
  drafts: RewardDraft[];
  onChange: (drafts: RewardDraft[]) => void;
  items: AdminItem[];
  currencies: Currency[];
  /** 用于显示分数线对应的段位 */
  tiers: Tier[];
  disabled?: boolean;
}

/**
 * 编辑赛季奖励: 每档一条分数线和一组货币、物品
 */
export function RewardsEditor({drafts, onChange, items, currencies, tiers, disabled}: RewardsEditorProps) {
  const update = (key: string, patch: Partial<RewardDraft>) =>
    onChange(drafts.map((d) => (d.key === key ? {...d, ...patch} : d)));

  return (
    <div className="flex flex-col gap-3">
      {drafts.map((d, i) => {
        const rating = parseAmount(d.minRating);
        return (
          <div key={d.key} className="flex flex-col gap-3 rounded-lg border p-3">
            <div className="flex items-center gap-2">
              <span className="shrink-0 text-sm font-medium">第 {i + 1} 档</span>
              <span className="shrink-0 text-sm text-muted-foreground">最高段位分 ≥</span>
              <Input
                inputMode="numeric"
                aria-label="分数线"
                className="w-28 font-mono tabular-nums"
                value={d.minRating}
                onChange={(e) => update(d.key, {minRating: e.target.value})}
                disabled={disabled}
              />
              {rating !== null && <TierName tier={tierOf(tiers, rating)} className="text-sm" />}
              <Button
                type="button"
                variant="ghost"
                size="icon"
                title="删除这一档"
                aria-label="删除这一档"
                className="ml-auto hover:text-destructive"
                onClick={() => onChange(drafts.filter((x) => x.key !== d.key))}
                disabled={disabled}
              >
                <Trash2 />
              </Button>
            </div>
            <BundleEditor
              drafts={d.bundle}
              onChange={(bundle) => update(d.key, {bundle})}
              items={items}
              currencies={currencies}
              max={MAX_REWARD_LINES}
              empty="还没有奖励内容"
              disabled={disabled}
            />
          </div>
        );
      })}
      {drafts.length === 0 && <p className="text-sm text-muted-foreground">没有赛季奖励</p>}

      <Button
        type="button"
        variant="outline"
        size="sm"
        className="self-start"
        onClick={() => onChange([...drafts, newRewardDraft(drafts)])}
        disabled={disabled || drafts.length >= MAX_REWARDS}
      >
        <Plus />
        添加一档
      </Button>
    </div>
  );
}
