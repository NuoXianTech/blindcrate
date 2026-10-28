'use client';

import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Textarea} from '@/components/ui/textarea';
import {CurrencyIcon} from '@/components/common/wallet';
import type {Currency} from '@/lib/services/admin';
import {InputField, SwitchField} from './Fields';
import type {FormTabProps} from './form';

/**
 * 基本信息: 代码、名称、描述、封面、排序、出价货币和开关
 */
export function BasicTab({form, onChange, disabled, editing, currencies}: FormTabProps & {
  /** 编辑时代码不能修改 */
  editing: boolean;
  currencies: Currency[];
}) {
  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <InputField
          id="venue-code"
          label="代码"
          help={editing ? '拼进玩法代码和统计范围, 创建后不能修改' : '1-24 个小写字母、数字或下划线, 创建后不能修改'}
          placeholder="如 old_town"
          maxLength={24}
          value={form.code}
          onChange={(code) => onChange({code})}
          disabled={disabled || editing}
        />
        <InputField
          id="venue-name"
          label="名称"
          maxLength={64}
          value={form.name}
          onChange={(name) => onChange({name})}
          disabled={disabled}
        />
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="venue-description">描述</Label>
        <Textarea
          id="venue-description"
          maxLength={255}
          placeholder="选填, 大厅中展示, 最多 255 个字符"
          value={form.description}
          onChange={(e) => onChange({description: e.target.value})}
          disabled={disabled}
        />
      </div>

      <div className="grid gap-3 sm:grid-cols-[1fr_8rem]">
        <InputField
          id="venue-image"
          label="封面图片"
          placeholder="选填, http(s) 地址"
          maxLength={1024}
          value={form.imageURL}
          onChange={(imageURL) => onChange({imageURL})}
          disabled={disabled}
        />
        <InputField
          id="venue-sort"
          label="排序"
          help="从小到大排列"
          inputMode="numeric"
          value={form.sortOrder}
          onChange={(sortOrder) => onChange({sortOrder})}
          disabled={disabled}
        />
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="venue-currency">出价货币</Label>
        <Select value={form.bidCurrency} onValueChange={(bidCurrency) => onChange({bidCurrency})} disabled={disabled}>
          <SelectTrigger id="venue-currency" className="w-full sm:w-60">
            <SelectValue placeholder="选择货币" />
          </SelectTrigger>
          <SelectContent>
            {currencies.map((c) => (
              <SelectItem key={c.code} value={c.code}>
                <CurrencyIcon currency={c.code} />
                {c.name}
                {!c.is_enabled && ' (已停用)'}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className="text-xs text-muted-foreground">出价、带入资金和仓库价值都按这个货币计算, 其他货币的藏品不计入仓库价值</span>
      </div>

      <div className="flex flex-col gap-2">
        <SwitchField
          id="venue-enabled"
          label="启用"
          help="停用后不能加入队列, 进行中的对局照常打完"
          checked={form.isEnabled}
          onChange={(isEnabled) => onChange({isEnabled})}
          disabled={disabled}
        />
        <SwitchField
          id="venue-pve"
          label="人机模式"
          help="1 个玩家和 3 个人机"
          checked={form.pveEnabled}
          onChange={(pveEnabled) => onChange({pveEnabled})}
          disabled={disabled}
        />
        <SwitchField
          id="venue-ranked"
          label="排位模式"
          help="至少 2 个玩家, 不够时用人机补齐"
          checked={form.rankedEnabled}
          onChange={(rankedEnabled) => onChange({rankedEnabled})}
          disabled={disabled}
        />
      </div>
    </div>
  );
}
