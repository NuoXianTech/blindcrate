'use client';

import {useState} from 'react';
import {Search} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {ALL, parseID, type IDFilters} from './constants';

/**
 * 用户、操作人等数字 ID 的筛选表单, 提交时才生效
 */
export function IDFilterForm({
  fields,
  values,
  onSubmit,
}: {
  fields: { key: string; label: string }[];
  values: IDFilters;
  onSubmit: (values: IDFilters) => void;
}) {
  const [inputs, setInputs] = useState<Record<string, string>>(() =>
    Object.fromEntries(fields.map((f) => [f.key, values[f.key]?.toString() ?? ''])),
  );

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const next: IDFilters = {};
    for (const f of fields) {
      const value = inputs[f.key].trim();
      if (value && !/^\d+$/.test(value)) {
        toast.error(`${f.label}只能填数字`);
        return;
      }
      next[f.key] = parseID(value);
    }
    onSubmit(next);
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-1 flex-wrap gap-2">
      {fields.map((f) => (
        <Input
          key={f.key}
          inputMode="numeric"
          placeholder={f.label}
          aria-label={f.label}
          value={inputs[f.key]}
          onChange={(e) => setInputs((prev) => ({...prev, [f.key]: e.target.value}))}
          className="w-28"
        />
      ))}
      <Button type="submit" variant="secondary">
        <Search />
        筛选
      </Button>
    </form>
  );
}

/**
 * 业务类型下拉框
 */
export function BizTypeSelect({value, labels, onChange}: { value: string; labels: Record<string, string>; onChange: (value: string) => void }) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="sm:w-36">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL}>全部类型</SelectItem>
        {Object.entries(labels).map(([v, label]) => (
          <SelectItem key={v} value={v}>
            {label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/**
 * 用户或操作人; 用户已不存在时只有 ID, 操作人为 0 表示系统操作
 */
export function Actor({id, username, system}: { id: number; username: string; system?: boolean }) {
  if (system && id === 0) return <span className="text-muted-foreground">系统</span>;
  return (
    <span className="flex flex-col">
      <span>{username ? `@${username}` : '已删除用户'}</span>
      <span className="text-xs text-muted-foreground">#{id}</span>
    </span>
  );
}
