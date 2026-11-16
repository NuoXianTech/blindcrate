'use client';

import {Plus, Trash2} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import type {ConditionType} from '@/lib/services/condition';
import type {AdminItem} from '@/lib/services/item';
import type {Currency} from '@/lib/services/wallet';
import {MAX_PARAM_ITEMS, newConditionDraft, type ConditionDraft} from './drafts';
import {ParamField} from './ParamField';

/**
 * 条件编辑器的Props接口
 */
interface ConditionEditorProps {
  drafts: ConditionDraft[];
  onChange: (drafts: ConditionDraft[]) => void;
  /** 后端提供的条件类型, 据此生成参数表单, 下拉框的可选值已经填好 */
  types: ConditionType[];
  /** 物品和货币目录, 只有带奖励参数的条件用到 */
  items: AdminItem[];
  currencies: Currency[];
  max: number;
  /** 没有任何条件时的提示 */
  empty: string;
  /** 嵌套在另一个条件里: 不能再选带一组条件的类型, 最多嵌套一层 */
  nested?: boolean;
  disabled?: boolean;
}

/**
 * 编辑一组条件, 如地图的入场条件; 参数表单由条件类型的字段描述生成
 */
export function ConditionEditor({drafts, onChange, types: allTypes, items, currencies, max, empty, nested, disabled}: ConditionEditorProps) {
  const types = nested ? allTypes.filter((t) => !t.fields.some((f) => f.type === 'conditions')) : allTypes;
  const update = (key: string, patch: Partial<ConditionDraft>) =>
    onChange(drafts.map((d) => (d.key === key ? {...d, ...patch} : d)));
  const changeType = (draft: ConditionDraft, value: string) => {
    const type = types.find((t) => t.type === value);
    if (type) update(draft.key, {...newConditionDraft(type), key: draft.key});
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
                    renderConditions={nested ? undefined : (inner, setInner) => (
                      <ConditionEditor
                        drafts={inner}
                        onChange={setInner}
                        types={allTypes}
                        items={items}
                        currencies={currencies}
                        max={MAX_PARAM_ITEMS}
                        empty="还没有条件"
                        nested
                        disabled={disabled}
                      />
                    )}
                    disabled={disabled}
                  />
                ))}
              </div>
            )}
          </div>
        );
      })}
      {drafts.length === 0 && <p className="py-2 text-sm text-muted-foreground">{empty}</p>}

      <div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => types[0] && onChange([...drafts, newConditionDraft(types[0])])}
          disabled={disabled || drafts.length >= max || !types.length}
        >
          <Plus />
          添加条件
        </Button>
      </div>
    </div>
  );
}
