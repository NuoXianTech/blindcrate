'use client';

import {Plus, Trash2} from 'lucide-react';
import {nanoid} from 'nanoid';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import type {LootPoolListItem} from '@/lib/services/admin';
import {FieldGroup, InputField} from './Fields';
import {MAX_SLOTS} from './constants';
import type {FormTabProps, SlotDraft} from './form';

/** 档位的列宽, 表头和每行共用 */
const SLOT_COLUMNS = 'grid grid-cols-[1fr_4.5rem_4.5rem_6rem_2.25rem] items-center gap-2';

/**
 * 仓库: 格子大小、档位和重抽次数
 */
export function WarehouseTab({form, onChange, disabled, pools}: FormTabProps & { pools: LootPoolListItem[] }) {
  const updateSlot = (key: string, patch: Partial<SlotDraft>) =>
    onChange({slots: form.slots.map((s) => (s.key === key ? {...s, ...patch} : s))});
  const width = Number(form.gridWidth);
  const height = Number(form.gridHeight);

  return (
    <div className="flex flex-col gap-6">
      <FieldGroup title="格子" help="藏品按面积从大到小摆进格子, 摆不下时整仓重抽">
        <div className="grid grid-cols-3 gap-3">
          <InputField
            id="venue-grid-width"
            label="宽度"
            help="1-16 格"
            inputMode="numeric"
            value={form.gridWidth}
            onChange={(gridWidth) => onChange({gridWidth})}
            disabled={disabled}
          />
          <InputField
            id="venue-grid-height"
            label="高度"
            help="1-16 格"
            inputMode="numeric"
            value={form.gridHeight}
            onChange={(gridHeight) => onChange({gridHeight})}
            disabled={disabled}
          />
          <InputField
            id="venue-rerolls"
            label="重抽次数上限"
            help="超过后开局失败"
            inputMode="numeric"
            value={form.maxRerolls}
            onChange={(maxRerolls) => onChange({maxRerolls})}
            disabled={disabled}
          />
        </div>
        {width > 0 && height > 0 && (
          <p className="text-xs text-muted-foreground">
            共 {width * height} 格, 模拟时摆放失败率需不超过 1%
          </p>
        )}
      </FieldGroup>

      <FieldGroup title="档位" help={`每个档位按出现概率决定是否出现, 出现时从掉落池抽取若干次; 最多 ${MAX_SLOTS} 个`}>
        <div className="flex flex-col gap-2">
          <div className={`${SLOT_COLUMNS} px-1 text-xs text-muted-foreground`}>
            <span>掉落池</span>
            <span>最少次数</span>
            <span>最多次数</span>
            <span>出现概率</span>
            <span />
          </div>
          {form.slots.map((s) => (
            <SlotRow
              key={s.key}
              slot={s}
              pools={pools}
              onChange={(patch) => updateSlot(s.key, patch)}
              onRemove={() => onChange({slots: form.slots.filter((x) => x.key !== s.key)})}
              disabled={disabled}
            />
          ))}
          {form.slots.length === 0 && <p className="py-2 text-sm text-muted-foreground">至少需要一个档位</p>}
          <div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onChange({slots: [...form.slots, {key: nanoid(), pool: '', min: '1', max: '1', chance: '100'}]})}
              disabled={disabled || form.slots.length >= MAX_SLOTS}
            >
              <Plus />
              添加档位
            </Button>
          </div>
        </div>
      </FieldGroup>
    </div>
  );
}

/**
 * 档位行的Props接口
 */
interface SlotRowProps {
  slot: SlotDraft;
  pools: LootPoolListItem[];
  onChange: (patch: Partial<SlotDraft>) => void;
  onRemove: () => void;
  disabled: boolean;
}

/**
 * 一个档位: 掉落池、抽取次数范围和出现概率
 */
function SlotRow({slot, pools, onChange, onRemove, disabled}: SlotRowProps) {
  const known = pools.some((p) => p.code === slot.pool);
  return (
    <div className={SLOT_COLUMNS}>
      <Select value={slot.pool} onValueChange={(pool) => onChange({pool})} disabled={disabled}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="选择掉落池" />
        </SelectTrigger>
        <SelectContent>
          {slot.pool && !known && <SelectItem value={slot.pool}>不存在的掉落池 {slot.pool}</SelectItem>}
          {pools.map((p) => (
            <SelectItem key={p.id} value={p.code}>
              {p.name}
              <span className="font-mono text-xs text-muted-foreground">
                {p.code} · {p.entry_count} 个条目{!p.is_enabled && ' · 已停用'}
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Input
        inputMode="numeric"
        aria-label="最少次数"
        className="font-mono tabular-nums"
        value={slot.min}
        onChange={(e) => onChange({min: e.target.value})}
        disabled={disabled}
      />
      <Input
        inputMode="numeric"
        aria-label="最多次数"
        className="font-mono tabular-nums"
        value={slot.max}
        onChange={(e) => onChange({max: e.target.value})}
        disabled={disabled}
      />
      <div className="relative">
        <Input
          inputMode="decimal"
          aria-label="出现概率"
          className="pr-7 font-mono tabular-nums"
          value={slot.chance}
          onChange={(e) => onChange({chance: e.target.value})}
          disabled={disabled}
        />
        <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-muted-foreground">%</span>
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        title="删除档位"
        aria-label="删除档位"
        className="hover:text-destructive"
        onClick={onRemove}
        disabled={disabled}
      >
        <Trash2 />
      </Button>
    </div>
  );
}
