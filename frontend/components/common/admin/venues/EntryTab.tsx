'use client';

import {BundleEditor, ConditionEditor, type ScopeOption} from '@/components/common/admin/editors';
import type {AdminItem, ConditionType, Currency} from '@/lib/services/admin';
import {FieldGroup} from './Fields';
import {MAX_ENTRY_COST_LINES, MAX_REQUIREMENTS} from './constants';
import type {FormTabProps} from './form';

/**
 * 入场: 入场代价和入场条件
 */
export function EntryTab({form, onChange, disabled, items, currencies, types, scopes}: FormTabProps & {
  items: AdminItem[];
  currencies: Currency[];
  types: ConditionType[];
  scopes: ScopeOption[];
}) {
  return (
    <div className="flex flex-col gap-6">
      <FieldGroup title="入场代价" help={`开局时扣除, 练习局不扣; 最多 ${MAX_ENTRY_COST_LINES} 项`}>
        <BundleEditor
          drafts={form.entryCost}
          onChange={(entryCost) => onChange({entryCost})}
          items={items}
          currencies={currencies}
          max={MAX_ENTRY_COST_LINES}
          empty="免费入场"
          disabled={disabled}
        />
      </FieldGroup>
      <FieldGroup title="入场条件" help={`需要全部满足才能加入队列; 最多 ${MAX_REQUIREMENTS} 个`}>
        <ConditionEditor
          drafts={form.requirements}
          onChange={(requirements) => onChange({requirements})}
          types={types}
          items={items}
          currencies={currencies}
          scopes={scopes}
          max={MAX_REQUIREMENTS}
          empty="没有入场条件, 所有玩家都能加入"
          disabled={disabled}
        />
      </FieldGroup>
    </div>
  );
}
