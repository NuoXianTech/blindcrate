'use client';

import {Plus, Trash2} from 'lucide-react';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Switch} from '@/components/ui/switch';
import {Textarea} from '@/components/ui/textarea';
import {useQualities} from '@/components/common/site';
import type {ConditionField} from '@/lib/services/condition';
import type {AdminItem} from '@/lib/services/item';
import type {Currency} from '@/lib/services/wallet';
import {BundleEditor} from './BundleEditor';
import {MAX_BUNDLE_LINES, MAX_PARAM_ITEMS, type BundleDraft, type ConditionDraft, type ParamDraft} from './drafts';
import {OptionCombobox} from './OptionCombobox';

/** 可选值超过这么多时用可搜索的下拉框 */
const SEARCHABLE_FROM = 12;

/**
 * 参数字段的Props接口
 */
export interface ParamFieldProps {
  id: string;
  field: ConditionField;
  value: ParamDraft | undefined;
  onChange: (value: ParamDraft) => void;
  /** 物品和货币目录, 只有结构化的奖励参数用到 */
  items: AdminItem[];
  currencies: Currency[];
  /** 渲染一组嵌套的条件, 由条件编辑器提供; 不提供时 conditions 参数不能编辑 */
  renderConditions?: (drafts: ConditionDraft[], onChange: (drafts: ConditionDraft[]) => void) => React.ReactNode;
  disabled?: boolean;
}

/**
 * 一个参数: 按字段的 input 选择输入控件, 基点参数按百分比输入; 条件、物品的使用效果和人机风格共用
 * 下拉框的可选值由后端填好, 这里不区分是货币、物品还是别的, 后端新增可选值来源时不用改这里
 */
export function ParamField({id, field, value, onChange, items, currencies, renderConditions, disabled}: ParamFieldProps) {
  const {qualityOf} = useQualities();
  const text = typeof value === 'string' ? value : '';
  const options = field.options ?? [];

  if (field.input === 'switch') {
    return (
      <div className="flex items-center justify-between gap-4 rounded-lg border p-3 sm:col-span-2">
        <div className="flex flex-col gap-0.5">
          <Label htmlFor={id}>{field.label}</Label>
          {field.help && <span className="text-xs text-muted-foreground">{field.help}</span>}
        </div>
        <Switch id={id} checked={value === true} onCheckedChange={onChange} disabled={disabled} />
      </div>
    );
  }

  let control: React.ReactNode;
  // 结构化的参数和多选占整行
  let wide = false;
  if (field.input === 'select') {
    // 不能新选的可选值只在已经选了时保留
    const choices = options.filter((o) => !o.disabled || o.value === text);
    control = choices.length >= SEARCHABLE_FROM ? (
      <OptionCombobox id={id} options={choices} value={text} onChange={onChange} placeholder={`选择${field.label}`} disabled={disabled} />
    ) : (
      <Select value={text} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue placeholder={`选择${field.label}`} />
        </SelectTrigger>
        <SelectContent>
          {text && !choices.some((o) => o.value === text) && <SelectItem value={text}>未知选项 {text}</SelectItem>}
          {choices.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              <span className={cn(o.quality && qualityOf(o.quality).text)}>{o.label}</span>
              {o.hint && <span className="text-xs text-muted-foreground">{o.hint}</span>}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  } else if (field.input === 'suggest') {
    control = (
      <>
        <Input id={id} list={`${id}-options`} className="font-mono" value={text} onChange={(e) => onChange(e.target.value)} disabled={disabled} />
        <datalist id={`${id}-options`}>
          {options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </datalist>
      </>
    );
  } else if (field.input === 'multiselect') {
    wide = true;
    const picked = Array.isArray(value) ? (value as string[]) : [];
    const toggle = (v: string) => onChange(picked.includes(v) ? picked.filter((x) => x !== v) : [...picked, v]);
    control = (
      <div id={id} role="group" aria-label={field.label} className="flex flex-wrap gap-1.5">
        {options.map((o) => (
          <Button
            key={o.value}
            type="button"
            size="sm"
            variant={picked.includes(o.value) ? 'default' : 'outline'}
            aria-pressed={picked.includes(o.value)}
            onClick={() => toggle(o.value)}
            disabled={disabled}
          >
            {o.label}
          </Button>
        ))}
      </div>
    );
  } else if (field.type === 'bundle') {
    wide = true;
    control = (
      <BundleEditor
        drafts={Array.isArray(value) ? (value as BundleDraft[]) : []}
        onChange={onChange}
        items={items}
        currencies={currencies}
        max={MAX_BUNDLE_LINES}
        empty="还没有内容"
        disabled={disabled}
      />
    );
  } else if (field.type === 'bundle_list') {
    wide = true;
    const groups = Array.isArray(value) ? (value as BundleDraft[][]) : [];
    control = (
      <div className="flex flex-col gap-2">
        {groups.map((drafts, i) => (
          <div key={i} className="flex flex-col gap-2 rounded-lg border p-3">
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm text-muted-foreground">第 {i + 1} 组</span>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                title="删除这一组"
                aria-label="删除这一组"
                className="hover:text-destructive"
                onClick={() => onChange(groups.filter((_, j) => j !== i))}
                disabled={disabled}
              >
                <Trash2 />
              </Button>
            </div>
            <BundleEditor
              drafts={drafts}
              onChange={(next) => onChange(groups.map((g, j) => (j === i ? next : g)))}
              items={items}
              currencies={currencies}
              max={MAX_BUNDLE_LINES}
              empty="还没有内容"
              disabled={disabled}
            />
          </div>
        ))}
        {groups.length === 0 && <p className="py-2 text-sm text-muted-foreground">还没有任何一组</p>}
        <div>
          <Button type="button" variant="outline" size="sm" onClick={() => onChange([...groups, []])} disabled={disabled || groups.length >= MAX_PARAM_ITEMS}>
            <Plus />
            添加一组
          </Button>
        </div>
      </div>
    );
  } else if (field.type === 'conditions') {
    wide = true;
    control = renderConditions ?
      renderConditions(Array.isArray(value) ? (value as ConditionDraft[]) : [], onChange) :
      <p className="text-sm text-muted-foreground">这里不能编辑条件</p>;
  } else if (field.input === 'textarea') {
    wide = true;
    control = <Textarea id={id} value={text} onChange={(e) => onChange(e.target.value)} disabled={disabled} />;
  } else if (field.input === 'datetime') {
    control = <Input id={id} type="datetime-local" value={text} onChange={(e) => onChange(e.target.value)} disabled={disabled} />;
  } else if (field.unit === 'bp') {
    control = (
      <div className="relative">
        <Input
          id={id}
          inputMode="decimal"
          className="pr-8 font-mono"
          value={text}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
        />
        <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-muted-foreground">%</span>
      </div>
    );
  } else {
    control = (
      <Input
        id={id}
        inputMode={field.input === 'number' ? 'numeric' : undefined}
        className={field.input === 'text' ? undefined : 'font-mono'}
        value={text}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      />
    );
  }

  return (
    <div className={cn('flex min-w-0 flex-col gap-2', wide && 'col-span-full')}>
      <Label htmlFor={id}>{field.label}</Label>
      {control}
      {field.help && <span className="text-xs text-muted-foreground">{field.help}</span>}
    </div>
  );
}
