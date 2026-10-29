'use client';

import {useId} from 'react';
import {Plus, Trash2} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import type {AdminItem, ConditionType, Currency} from '@/lib/services/admin';
import {newConditionDraft, type ConditionDraft} from './drafts';
import {ParamField} from './ParamField';

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
