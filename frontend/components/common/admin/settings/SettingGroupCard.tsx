'use client';

import {useState} from 'react';
import {useRouter} from 'next/navigation';
import {Undo2} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {SettingGroup, UpdateSettingRequest} from '@/lib/services/admin';
import {refreshPublicSettings} from '@/lib/site-actions';
import {formatDateTime} from '@/lib/utils';
import {MailTest} from './MailTest';
import {SettingInput} from './SettingInput';
import {diff, toFormValues} from './constants';

/**
 * 一个设置组的卡片, 只提交改动的字段
 */
export function SettingGroupCard({group, onSaved}: { group: SettingGroup; onSaved: (group: SettingGroup) => void }) {
  const router = useRouter();
  const {pending, submit} = useSubmit();
  const [values, setValues] = useState(() => toFormValues(group));
  // 要清空的密钥字段
  const [clearing, setClearing] = useState<string[]>([]);
  const changes = diff(group, values);
  const dirty = Object.keys(changes).length > 0 || clearing.length > 0;

  const reset = (g: SettingGroup) => {
    setValues(toFormValues(g));
    setClearing([]);
  };

  const setValue = (key: string, value: string | boolean) => setValues((prev) => ({...prev, [key]: value}));

  const toggleClear = (key: string) => {
    setValue(key, '');
    setClearing((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
  };

  const handleSubmit = submit(async () => {
    const body: UpdateSettingRequest = {value: {}, clear_secrets: clearing};
    for (const f of group.fields) {
      if (!(f.key in changes)) continue;
      const v = changes[f.key];
      if (f.type === 'int') {
        check(/^-?\d+$/.test(v as string) ? null : `${f.label}需为整数`);
        body.value[f.key] = Number(v);
      } else {
        body.value[f.key] = v;
      }
    }
    const updated = await services.admin.updateSetting(group.key, body);
    toast.success(`已保存${group.label}设置`);
    reset(updated);
    onSaved(updated);
    // 有公开字段时刷新站点缓存, 站点名称、公告等立即生效
    if (group.fields.some((f) => f.public)) {
      await refreshPublicSettings();
      router.refresh();
    }
  });

  return (
    <section className="flex flex-col gap-6 rounded-xl border bg-card p-6">
      <div className="flex flex-col justify-between gap-2 sm:flex-row sm:items-start">
        <div className="flex flex-col gap-1">
          <h2 className="text-lg font-semibold">{group.label}</h2>
          <p className="text-sm text-muted-foreground">{group.description}</p>
        </div>
        <span className="shrink-0 text-xs text-muted-foreground">
          {group.updated_at ? `上次保存于 ${formatDateTime(group.updated_at)}` : '尚未保存过, 使用默认值'}
        </span>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-6" noValidate>
        <div className="grid gap-4 sm:grid-cols-2">
          {group.fields.map((f) => (
            <SettingInput
              key={f.key}
              field={f}
              id={`setting-${group.key}-${f.key}`}
              value={values[f.key]}
              onChange={(v) => setValue(f.key, v)}
              disabled={pending}
              secretSet={!!group.secrets_set[f.key]}
              secretBroken={!!group.secrets_broken?.includes(f.key)}
              clearing={clearing.includes(f.key)}
              onToggleClear={() => toggleClear(f.key)}
            />
          ))}
        </div>
        <div className="flex items-center justify-end gap-2">
          {dirty && <span className="mr-auto text-xs text-muted-foreground">有未保存的修改</span>}
          <Button type="button" variant="outline" onClick={() => reset(group)} disabled={pending || !dirty}>
            <Undo2 />
            撤销修改
          </Button>
          <SubmitButton pending={pending} disabled={!dirty}>
            保存
          </SubmitButton>
        </div>
      </form>

      {group.key === 'mail' && <MailTest dirty={dirty} />}
    </section>
  );
}
