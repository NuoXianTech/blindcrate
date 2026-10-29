'use client';

import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Switch} from '@/components/ui/switch';
import {ItemCombobox} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import {CurrencyIcon} from '@/components/common/wallet';
import type {AdminItem, ConditionField, Currency} from '@/lib/services/admin';

/**
 * 参数字段的Props接口
 */
export interface ParamFieldProps {
  id: string;
  field: ConditionField;
  value: string | boolean | undefined;
  onChange: (value: string | boolean) => void;
  items: AdminItem[];
  currencies: Currency[];
  /** 统计范围候选项的 datalist ID, 没有 scope 字段时可以不传 */
  scopeListID?: string;
  disabled?: boolean;
}

/**
 * 一个参数: 按字段的 input 选择输入控件, 条件和物品的使用效果共用
 */
export function ParamField({id, field, value, onChange, items, currencies, scopeListID, disabled}: ParamFieldProps) {
  const {qualities} = useQualities();
  const text = typeof value === 'string' ? value : '';

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
  if (field.input === 'metric') {
    const options = field.options ?? [];
    control = (
      <Select value={text} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue placeholder="选择统计数据" />
        </SelectTrigger>
        <SelectContent>
          {text && !options.some((o) => o.value === text) && <SelectItem value={text}>未知统计数据 {text}</SelectItem>}
          {options.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  } else if (field.input === 'quality') {
    control = (
      <Select value={text} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue placeholder="选择品质" />
        </SelectTrigger>
        <SelectContent>
          {text && !qualities.some((q) => String(q.level) === text) && <SelectItem value={text}>未知品质 {text}</SelectItem>}
          {qualities.map((q) => (
            <SelectItem key={q.level} value={String(q.level)}>
              <span className={q.text}>{q.name}</span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  } else if (field.input === 'currency') {
    control = (
      <Select value={text} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue placeholder="选择货币" />
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
    );
  } else if (field.input === 'item') {
    const itemID = text ? Number(text) : null;
    const item = itemID !== null ? items.find((i) => i.id === itemID) : undefined;
    control = (
      <ItemCombobox
        options={items
            .filter((i) => i.id === itemID || i.is_enabled)
            .map((i) => ({id: i.id, name: i.name, quality: i.quality}))}
        value={itemID}
        onChange={(id) => onChange(String(id))}
        placeholder={itemID !== null && !item ? `已删除的物品 #${itemID}` : '选择物品'}
        disabled={disabled}
      />
    );
  } else {
    control = (
      <Input
        id={id}
        list={field.input === 'scope' ? scopeListID : undefined}
        inputMode={field.input === 'number' ? 'numeric' : undefined}
        className={field.input === 'text' ? undefined : 'font-mono'}
        value={text}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      />
    );
  }

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <Label htmlFor={id}>{field.label}</Label>
      {control}
      {field.help && <span className="text-xs text-muted-foreground">{field.help}</span>}
    </div>
  );
}
