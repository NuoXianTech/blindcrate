'use client';

import {cn} from 'cn';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {useQualities} from '@/components/common/site';
import {HOST_PERSONAS} from '@/lib/auction';
import {FieldGroup, InputField, SwitchField} from './Fields';
import {MAX_INTEL} from './constants';
import type {FormTabProps} from './form';

/** 不区分高品质时的选项值 */
const NO_QUALITY = '0';

/**
 * 主持人: 每轮的提示、性格和撒谎率, 以及向主持人买情报的次数和估价情报的误差; 情报的价格按期望价值推导, 见拍卖页
 */
export function HostTab({form, onChange, disabled}: FormTabProps) {
  const {qualities} = useQualities();
  const togglePersona = (code: string) => {
    const next = form.hostPersonas.includes(code) ? form.hostPersonas.filter((p) => p !== code) : [...form.hostPersonas, code];
    onChange({hostPersonas: next});
  };

  return (
    <div className="flex flex-col gap-6">
      <SwitchField
        id="venue-host-enabled"
        label="主持人提示"
        help="开局时选出主持人的性格(开箱后才公开), 每轮开始时公开一条关于仓库的提示, 有一定概率说错, 开箱后公开真假; 热度随每轮的加价变化"
        checked={form.hostEnabled}
        onChange={(hostEnabled) => onChange({hostEnabled})}
        disabled={disabled}
      />

      <FieldGroup title="性格" help="开局时从选中的性格中随机一种, 都不选表示全部; 性格决定说错时往哪边错">
        <div className="grid gap-2 sm:grid-cols-3">
          {Object.entries(HOST_PERSONAS).map(([code, persona]) => {
            const active = form.hostPersonas.includes(code);
            return (
              <button
                key={code}
                type="button"
                className={cn(
                    'flex flex-col gap-0.5 rounded-lg border p-2.5 text-left text-sm transition-colors',
                  active ? 'border-primary bg-primary/5' : 'hover:bg-accent',
                  disabled && 'cursor-not-allowed opacity-50',
                )}
                aria-pressed={active}
                onClick={() => togglePersona(code)}
                disabled={disabled}
              >
                <span className="font-medium">{persona.label}</span>
                <span className="text-xs text-muted-foreground">{persona.description}</span>
              </button>
            );
          })}
        </div>
      </FieldGroup>

      <FieldGroup title="提示" help="撒谎率为基础值: 稳重的减半, 煽动的在热度不低于 70 时、狡黠的在热度不高于 30 时 × 1.5, 涉及高品质时再 × 1.5, 最多 25%">
        <div className="grid grid-cols-2 gap-3">
          <InputField
            id="venue-lie-rate"
            label="撒谎率"
            suffix="%"
            inputMode="decimal"
            value={form.lieRate}
            onChange={(lieRate) => onChange({lieRate})}
            disabled={disabled}
          />
          <div className="flex min-w-0 flex-col gap-2">
            <Label htmlFor="venue-high-quality">高品质</Label>
            <Select
              value={String(form.highQuality)}
              onValueChange={(v) => onChange({highQuality: Number(v)})}
              disabled={disabled}
            >
              <SelectTrigger id="venue-high-quality" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NO_QUALITY}>不区分</SelectItem>
                {form.highQuality > 0 && !qualities.some((q) => q.level === form.highQuality) && (
                  <SelectItem value={String(form.highQuality)}>未知品质 {form.highQuality}</SelectItem>
                )}
                {qualities.map((q) => (
                  <SelectItem key={q.level} value={String(q.level)}>
                    <span className={q.text}>{q.name}</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <span className="text-xs text-muted-foreground">也是&ldquo;高品质件数&rdquo;提示统计的最低品质, 不区分时取中间的品质</span>
          </div>
        </div>
      </FieldGroup>

      <FieldGroup
        title="情报"
        help="出价阶段向主持人买只有自己看得到的情报, 每轮最多一次: 前三分之一的轮次看一件藏品的品质, 中间看一件藏品的名称, 之后看仓库价值的区间; 价格见拍卖页, 从可用余额扣除"
      >
        <div className="grid grid-cols-2 gap-3">
          <InputField
            id="venue-intel-limit"
            label="每局上限"
            suffix="次"
            inputMode="numeric"
            help={`0 表示不卖情报, 最多 ${MAX_INTEL} 次`}
            value={form.intelLimit}
            onChange={(intelLimit) => onChange({intelLimit})}
            disabled={disabled}
          />
          <InputField
            id="venue-intel-precision"
            label="估价情报误差"
            suffix="%"
            inputMode="decimal"
            help="区间宽度为价值 × 误差 × 2, 5%-50%"
            value={form.intelPrecision}
            onChange={(intelPrecision) => onChange({intelPrecision})}
            disabled={disabled}
          />
        </div>
      </FieldGroup>
    </div>
  );
}
