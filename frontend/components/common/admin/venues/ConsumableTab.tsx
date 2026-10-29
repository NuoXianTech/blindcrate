'use client';

import {X} from 'lucide-react';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {ItemCombobox} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import type {AdminItem, ItemEffect} from '@/lib/services/admin';
import {FieldGroup, InputField} from './Fields';
import {MAX_ALLOWED_ITEMS, MAX_USES, SCENE_MATCH} from './constants';
import type {FormTabProps} from './form';

/**
 * 道具: 每个模式每局最多使用的道具数, 以及允许使用的道具
 */
export function ConsumableTab({form, onChange, disabled, items, effects}: FormTabProps & {
  items: AdminItem[];
  effects: ItemEffect[];
}) {
  const {qualityOf} = useQualities();
  const effectOf = (item: AdminItem) => effects.find((e) => e.code === item.effect);
  // 只有对局中使用的效果才能加入, 已下架的道具不能新加, 已选的照常显示
  const usable = items.filter((i) => i.is_enabled && effectOf(i)?.scene === SCENE_MATCH && !form.allowedItems.includes(i.id));

  return (
    <div className="flex flex-col gap-6">
      <FieldGroup title="每局使用上限" help={`0 表示这个模式不能用道具, 最多 ${MAX_USES} 个; 练习局不能用道具`}>
        <div className="grid grid-cols-2 gap-3">
          <InputField
            id="venue-uses-pve"
            label="人机"
            suffix="个"
            inputMode="numeric"
            value={form.usesPve}
            onChange={(usesPve) => onChange({usesPve})}
            disabled={disabled}
          />
          <InputField
            id="venue-uses-ranked"
            label="排位"
            suffix="个"
            inputMode="numeric"
            value={form.usesRanked}
            onChange={(usesRanked) => onChange({usesRanked})}
            disabled={disabled}
          />
        </div>
      </FieldGroup>

      <FieldGroup
        title="允许的道具"
        help={`为空表示所有对局中可用的道具都可以用, 否则只能用列出的道具; 最多 ${MAX_ALLOWED_ITEMS} 个`}
      >
        {form.allowedItems.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {form.allowedItems.map((id) => {
              const item = items.find((i) => i.id === id);
              const q = qualityOf(item?.quality ?? 0);
              return (
                <span key={id} className="inline-flex items-center gap-1 rounded-md border py-0.5 pr-0.5 pl-2 text-sm">
                  <span className={cn(item && q.text)}>{item?.name ?? `物品 #${id}`}</span>
                  {item && <span className="text-xs text-muted-foreground">{effectOf(item)?.label ?? '没有使用效果'}</span>}
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-6"
                    title="移除"
                    aria-label="移除"
                    onClick={() => onChange({allowedItems: form.allowedItems.filter((x) => x !== id)})}
                    disabled={disabled}
                  >
                    <X />
                  </Button>
                </span>
              );
            })}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">所有道具都可以用</p>
        )}
        <ItemCombobox
          options={usable}
          value={null}
          onChange={(id) => onChange({allowedItems: [...form.allowedItems, id]})}
          placeholder={usable.length > 0 ? '添加道具' : '没有可添加的道具'}
          disabled={disabled || usable.length === 0 || form.allowedItems.length >= MAX_ALLOWED_ITEMS}
        />
      </FieldGroup>
    </div>
  );
}
