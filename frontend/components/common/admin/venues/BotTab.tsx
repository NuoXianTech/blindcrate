'use client';

import {Plus, X} from 'lucide-react';
import {Button} from '@/components/ui/button';
import type {Currency} from '@/lib/services/admin';
import {FieldGroup, InputField, SwitchField} from './Fields';
import {BOT_LEVELS, MAX_ROUND_FACTORS} from './constants';
import type {FormTabProps} from './form';

/**
 * 人机: 人机的等级和出价参数, 人机模式的防刷参数
 */
export function BotTab({form, onChange, disabled, currency}: FormTabProps & { currency?: Currency }) {
  const setAt = (list: string[], i: number, value: string) => list.map((x, j) => (j === i ? value : x));

  return (
    <div className="flex flex-col gap-6">
      <FieldGroup title="等级" help="每个人机按权重随机一个等级, 估值 = 仓库价值 × (1 ± 误差), 误差越小越难对付">
        <div className="grid grid-cols-3 gap-3">
          {BOT_LEVELS.map((level, i) => (
            <div key={level} className="flex flex-col gap-2 rounded-lg border p-3">
              <span className="text-sm font-medium">{level}级</span>
              <InputField
                id={`venue-bot-weight-${i}`}
                label="权重"
                help="0-100"
                inputMode="numeric"
                value={form.levelWeights[i] ?? ''}
                onChange={(x) => onChange({levelWeights: setAt(form.levelWeights, i, x)})}
                disabled={disabled}
              />
              <InputField
                id={`venue-bot-error-${i}`}
                label="估值误差"
                help="0-90%"
                suffix="%"
                inputMode="decimal"
                value={form.estimateErrors[i] ?? ''}
                onChange={(x) => onChange({estimateErrors: setAt(form.estimateErrors, i, x)})}
                disabled={disabled}
              />
            </div>
          ))}
        </div>
      </FieldGroup>

      <FieldGroup title="出价" help="每轮出价 = 估值 × 本轮系数 × (1 ± 浮动), 不超过估值 × 出价上限; 轮数多于系数个数时沿用最后一个">
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-4">
          {form.roundFactors.map((value, i) => (
            <div key={i} className="flex items-end gap-1">
              <InputField
                id={`venue-round-factor-${i}`}
                label={`第 ${i + 1} 轮系数`}
                suffix="%"
                inputMode="decimal"
                value={value}
                onChange={(x) => onChange({roundFactors: setAt(form.roundFactors, i, x)})}
                disabled={disabled}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                title="删除"
                aria-label="删除"
                className="shrink-0 hover:text-destructive"
                onClick={() => onChange({roundFactors: form.roundFactors.filter((_, j) => j !== i)})}
                disabled={disabled || form.roundFactors.length <= 1}
              >
                <X />
              </Button>
            </div>
          ))}
        </div>
        <div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onChange({roundFactors: [...form.roundFactors, form.roundFactors.at(-1) ?? '80']})}
            disabled={disabled || form.roundFactors.length >= MAX_ROUND_FACTORS}
          >
            <Plus />
            添加一轮
          </Button>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <InputField
            id="venue-bot-jitter"
            label="出价浮动"
            help="0-50%"
            suffix="%"
            inputMode="decimal"
            value={form.jitter}
            onChange={(jitter) => onChange({jitter})}
            disabled={disabled}
          />
          <InputField
            id="venue-bot-max-factor"
            label="出价上限"
            help="估值的 1-100%"
            suffix="%"
            inputMode="decimal"
            value={form.maxFactor}
            onChange={(maxFactor) => onChange({maxFactor})}
            disabled={disabled}
          />
        </div>
      </FieldGroup>

      <FieldGroup title="诈唬" help="前几轮有一定概率按估值 × 诈唬系数出价, 用来干扰玩家判断">
        <div className="grid gap-3 sm:grid-cols-3">
          <InputField
            id="venue-bluff-chance"
            label="诈唬概率"
            help="0-100%"
            suffix="%"
            inputMode="decimal"
            value={form.bluffChance}
            onChange={(bluffChance) => onChange({bluffChance})}
            disabled={disabled}
          />
          <InputField
            id="venue-bluff-rounds"
            label="诈唬轮数"
            help="只在前几轮诈唬, 0-10"
            inputMode="numeric"
            value={form.bluffRounds}
            onChange={(bluffRounds) => onChange({bluffRounds})}
            disabled={disabled}
          />
          <InputField
            id="venue-bluff-factor"
            label="诈唬系数"
            help="估值的 1-100%"
            suffix="%"
            inputMode="decimal"
            value={form.bluffFactor}
            onChange={(bluffFactor) => onChange({bluffFactor})}
            disabled={disabled}
          />
        </div>
      </FieldGroup>

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
