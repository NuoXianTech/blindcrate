'use client';

import {useId} from 'react';
import {Plus, Trash2} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Switch} from '@/components/ui/switch';
import {ItemCombobox} from '@/components/common/item';
import {CurrencyIcon} from '@/components/common/wallet';
import type {AdminItem, ConditionField, ConditionType, Currency} from '@/lib/services/admin';
import {newConditionDraft, type ConditionDraft} from './drafts';

/** 统计范围的候选项 */
export interface ScopeOption {
  value: string;
  label: string;
}

/**
 * 条件编辑器的Props接口
 */
interface ConditionEditorProps {
  drafts: ConditionDraft[];
  onChange: (drafts: ConditionDraft[]) => void;
  /** 后端提供的条件类型, 据此生成参数表单 */
  types: ConditionType[];
  items: AdminItem[];
  currencies: Currency[];
  /** 统计范围输入框的候选项, 也可以手动输入 */
  scopes: ScopeOption[];
  max: number;
  /** 没有任何条件时的提示 */
  empty: string;
  disabled?: boolean;
}

/**
 * 编辑一组条件, 如地图的入场条件; 参数表单由条件类型的字段描述生成
 */
export function ConditionEditor({drafts, onChange, types, items, currencies, scopes, max, empty, disabled}: ConditionEditorProps) {
  const scopeListID = useId();
  const update = (key: string, patch: Partial<ConditionDraft>) =>
    onChange(drafts.map((d) => (d.key === key ? {...d, ...patch} : d)));
  const changeType = (draft: ConditionDraft, value: string) => {
    const type = types.find((t) => t.type === value);
    if (type) update(draft.key, {...newConditionDraft(type, currencies), key: draft.key});
  };

  return (
    <div className="flex flex-col gap-2">
      {drafts.map((d, i) => {
        const type = types.find((t) => t.type === d.type);
        return (
          <div key={d.key} className="flex flex-col gap-3 rounded-lg border p-3">
            <div className="flex items-center gap-2">
              <span className="shrink-0 text-sm text-muted-foreground">{i + 1}.</span>
              <Select value={d.type} onValueChange={(value) => changeType(d, value)} disabled={disabled}>
                <SelectTrigger className="flex-1">
                  <SelectValue placeholder="选择条件类型" />
                </SelectTrigger>
                <SelectContent>
                  {!type && d.type && <SelectItem value={d.type}>未知类型 {d.type}</SelectItem>}
                  {types.map((t) => (
                    <SelectItem key={t.type} value={t.type}>
                      {t.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                title="删除条件"
                aria-label="删除条件"
                className="hover:text-destructive"
                onClick={() => onChange(drafts.filter((x) => x.key !== d.key))}
                disabled={disabled}
              >
                <Trash2 />
              </Button>
            </div>
            {type?.description && <p className="text-xs text-muted-foreground">{type.description}</p>}
            {type && type.fields.length > 0 && (
              <div className="grid gap-3 sm:grid-cols-2">
                {type.fields.map((field) => (
                  <ParamField
                    key={field.key}
                    id={`${d.key}-${field.key}`}
                    field={field}
                    value={d.params[field.key]}
                    onChange={(value) => update(d.key, {params: {...d.params, [field.key]: value}})}
                    items={items}
                    currencies={currencies}
                    scopeListID={scopeListID}
                    disabled={disabled}
                  />
                ))}
              </div>
            )}
          </div>
        );
      })}
      {drafts.length === 0 && <p className="py-2 text-sm text-muted-foreground">{empty}</p>}

      <datalist id={scopeListID}>
        {scopes.map((s) => (
          <option key={s.value} value={s.value}>
            {s.label}
          </option>
        ))}
      </datalist>

      <div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => types[0] && onChange([...drafts, newConditionDraft(types[0], currencies)])}
          disabled={disabled || drafts.length >= max || !types.length}
        >
          <Plus />
          添加条件
        </Button>
      </div>
    </div>
  );
}

/**
 * 参数字段的Props接口
 */
interface ParamFieldProps {
  id: string;
  field: ConditionField;
  value: string | boolean | undefined;
  onChange: (value: string | boolean) => void;
  items: AdminItem[];
  currencies: Currency[];
  scopeListID: string;
  disabled?: boolean;
}

/**
 * 一个参数: 按字段的 input 选择输入控件
 */
function ParamField({id, field, value, onChange, items, currencies, scopeListID, disabled}: ParamFieldProps) {
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
  if (field.input === 'currency') {
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
