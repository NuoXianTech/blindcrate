'use client';

import {Plus, Trash2} from 'lucide-react';
import {nanoid} from 'nanoid';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import type {Currency} from '@/lib/services/admin';
import type {BotPersona} from '@/lib/services/bot';
import {FieldGroup, InputField, SwitchField} from './Fields';
import {MAX_BOT_PERSONAS} from './constants';
import type {FormTabProps, WeightDraft} from './form';

/** 出场风格的列宽, 表头和每行共用 */
const WEIGHT_COLUMNS = 'grid grid-cols-[1fr_5rem_4rem_2.25rem] items-center gap-2';

/**
 * 人机: 每个模式出场的人机风格和权重, 人机模式的防刷参数
 */
export function BotTab({form, onChange, disabled, currency, personas}: FormTabProps & {
  currency?: Currency;
  personas: BotPersona[];
}) {
  return (
    <div className="flex flex-col gap-6">
      <WeightList
        mode="人机"
        enabled={form.pveEnabled}
        drafts={form.botsPve}
        personas={personas}
        onChange={(botsPve) => onChange({botsPve})}
        disabled={disabled}
      />
      <WeightList
        mode="排位"
        enabled={form.rankedEnabled}
        drafts={form.botsRanked}
        personas={personas}
        onChange={(botsRanked) => onChange({botsRanked})}
        disabled={disabled}
      />

      <FieldGroup title="人机模式" help="防止玩家反复打人机刷收益; 全局的人机每日收益上限另外生效">
        <SwitchField
          id="venue-practice-only"
          label="只能练习"
          help="人机模式只能打练习局, 不扣入场代价, 也不结算收益"
          checked={form.practiceOnly}
          onChange={(practiceOnly) => onChange({practiceOnly})}
          disabled={disabled}
        />
        <div className="grid gap-3 sm:grid-cols-2">
          <InputField
            id="venue-daily-cap"
            label="每日收益上限"
            help="当天在这张地图的净收益达到后转为练习局, 0 表示不单独限制"
            suffix={currency?.name}
            inputMode={currency && currency.decimals > 0 ? 'decimal' : 'numeric'}
            value={form.dailyProfitCap}
            onChange={(dailyProfitCap) => onChange({dailyProfitCap})}
            disabled={disabled}
          />
          <InputField
            id="venue-value-factor"
            label="仓库价值系数"
            help="按这个比例缩放各档位的抽取次数, 10-100%"
            suffix="%"
            inputMode="decimal"
            value={form.valueFactor}
            onChange={(valueFactor) => onChange({valueFactor})}
            disabled={disabled}
          />
        </div>
      </FieldGroup>
    </div>
  );
}

/** 输入框里的权重, 格式不对或超出范围时按 0 计算出场概率 */
function weightOf(text: string): number {
  const weight = Number(text.trim());
  return Number.isInteger(weight) && weight >= 1 && weight <= 100 ? weight : 0;
}

/**
 * 一个模式的出场风格列表的Props接口
 */
interface WeightListProps {
  /** 模式名称, 如人机 */
  mode: string;
  /** 开放的模式至少配置一种风格 */
  enabled: boolean;
  drafts: WeightDraft[];
  personas: BotPersona[];
  onChange: (drafts: WeightDraft[]) => void;
  disabled: boolean;
}

/**
 * 一个模式出场的人机风格, 开局时每个人机按权重随机选一种
 */
function WeightList({mode, enabled, drafts, personas, onChange, disabled}: WeightListProps) {
  const update = (key: string, patch: Partial<WeightDraft>) =>
    onChange(drafts.map((d) => (d.key === key ? {...d, ...patch} : d)));
  const weights = drafts.map((d) => weightOf(d.weight));
  const total = weights.reduce((sum, w) => sum + w, 0);
  // 新加的一行默认选第一个还没用到的风格
  const unused = personas.find((p) => !drafts.some((d) => d.persona === p.code));

  return (
    <FieldGroup
      title={`${mode}模式`}
      help={
        enabled ?
          `开局时每个人机按权重随机选一种风格, 权重 1-100; 最多 ${MAX_BOT_PERSONAS} 种` :
          '这个模式没有开放, 开放后按这里的配置出场'
      }
    >
      <div className="flex flex-col gap-2">
        {drafts.length > 0 && (
          <div className={`${WEIGHT_COLUMNS} px-1 text-xs text-muted-foreground`}>
            <span>风格</span>
            <span>权重</span>
            <span className="text-right">出场概率</span>
            <span />
          </div>
        )}
        {drafts.map((d, i) => (
          <WeightRow
            key={d.key}
            draft={d}
            share={total > 0 ? weights[i] / total : 0}
            personas={personas}
            onChange={(patch) => update(d.key, patch)}
            onRemove={() => onChange(drafts.filter((x) => x.key !== d.key))}
            disabled={disabled}
          />
        ))}
        {drafts.length === 0 && (
          <p className="py-2 text-sm text-muted-foreground">{enabled ? '至少需要一种风格' : '没有配置风格'}</p>
        )}
        <div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onChange([...drafts, {key: nanoid(), persona: unused?.code ?? '', weight: '10'}])}
            disabled={disabled || drafts.length >= MAX_BOT_PERSONAS}
          >
            <Plus />
            添加风格
          </Button>
        </div>
      </div>
    </FieldGroup>
  );
}

/**
 * 出场风格行的Props接口
 */
interface WeightRowProps {
  draft: WeightDraft;
  /** 按当前权重算出的出场概率, 0-1 */
  share: number;
  personas: BotPersona[];
  onChange: (patch: Partial<WeightDraft>) => void;
  onRemove: () => void;
  disabled: boolean;
}

/**
 * 一种出场的风格: 风格、权重和出场概率
 */
function WeightRow({draft, share, personas, onChange, onRemove, disabled}: WeightRowProps) {
  const known = personas.some((p) => p.code === draft.persona);
  return (
    <div className={WEIGHT_COLUMNS}>
      <Select value={draft.persona} onValueChange={(persona) => onChange({persona})} disabled={disabled}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="选择风格" />
        </SelectTrigger>
        <SelectContent>
          {draft.persona && !known && <SelectItem value={draft.persona}>不存在的风格 {draft.persona}</SelectItem>}
          {personas.map((p) => (
            <SelectItem key={p.id} value={p.code}>
              {p.name}
              <span className="font-mono text-xs text-muted-foreground">{p.code}</span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Input
        inputMode="numeric"
        aria-label="权重"
        className="font-mono tabular-nums"
        value={draft.weight}
        onChange={(e) => onChange({weight: e.target.value})}
        disabled={disabled}
      />
      <span className="text-right font-mono text-sm tabular-nums text-muted-foreground">{Math.round(share * 100)}%</span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        title="删除风格"
        aria-label="删除风格"
        className="hover:text-destructive"
        onClick={onRemove}
        disabled={disabled}
      >
        <Trash2 />
      </Button>
    </div>
  );
}
