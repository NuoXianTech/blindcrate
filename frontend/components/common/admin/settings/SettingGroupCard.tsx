'use client';

import {useState} from 'react';
import {useRouter} from 'next/navigation';
import {RotateCcw, Undo2} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {SettingGroup, UpdateSettingRequest} from '@/lib/services/settings';
import {refreshPublicSettings} from '@/lib/site-actions';
import {formatDateTime} from '@/lib/utils';
import {MailTest} from './MailTest';
import {ResetSettingDialog} from './ResetSettingDialog';
import {SettingInput} from './SettingInput';
import {diff, type FormValue, intError, parseIntInput, toFormValues} from './constants';

/**
 * 一个设置组的卡片, 只提交改动的字段; 可以恢复默认值
 */
export function SettingGroupCard({group, onSaved}: { group: SettingGroup; onSaved: (group: SettingGroup) => void }) {
  const router = useRouter();
  const {pending, submit} = useSubmit();
  const [values, setValues] = useState(() => toFormValues(group));
  // 要清空的密钥字段
  const [clearing, setClearing] = useState<string[]>([]);
  const [resetting, setResetting] = useState(false);
  const changes = diff(group, values);
  const dirty = Object.keys(changes).length > 0 || clearing.length > 0;

  const reset = (g: SettingGroup) => {
    setValues(toFormValues(g));
    setClearing([]);
  };

  const setValue = (key: string, value: FormValue) => setValues((prev) => ({...prev, [key]: value}));

  const toggleClear = (key: string) => {
    setValue(key, '');
    setClearing((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
  };

  // 保存、恢复默认值成功后: 表单换成保存后的值, 有公开字段时刷新站点缓存, 站点名称、公告等立即生效
  const applySaved = async (updated: SettingGroup) => {
    reset(updated);
    onSaved(updated);
    // 没有任何字段变化时后端不保存, 保存时间不变
    if (updated.updated_at !== group.updated_at && group.fields.some((f) => f.public)) {
      // 设置已经保存, 刷新缓存失败(如页面在发布新版本前打开, 找不到服务端函数)不算保存失败, 缓存过一分钟也会自动更新
      await refreshPublicSettings().catch(() => undefined);
      router.refresh();
    }
  };

  const handleSubmit = submit(async () => {
    const body: UpdateSettingRequest = {value: {}, clear_secrets: clearing};
    for (const f of group.fields) {
      if (!(f.key in changes)) continue;
      const v = changes[f.key];
      if (f.type === 'int') {
        const n = parseIntInput(f, v as string);
        check(n === null ? intError(f) : null);
        body.value[f.key] = n as number;
      } else if (f.type === 'int_list') {
        body.value[f.key] = (v as string[]).map((s, i) => {
          const n = parseIntInput(f, s);
          check(n === null ? intError(f, i) : null);
          return n as number;
        });
      } else {
        body.value[f.key] = v as string | boolean;
      }
    }
    const updated = await services.settingsAdmin.updateSetting(group.key, body);
    toast.success(`已保存${group.label}设置`);
    await applySaved(updated);
  });

  return (
    <section className="flex flex-col gap-6 rounded-xl border bg-card p-6">
      <div className="flex flex-col justify-between gap-2 sm:flex-row sm:items-start">
        <div className="flex flex-col gap-1">
          <h2 className="text-lg font-semibold">{group.label}</h2>
          <p className="text-sm text-muted-foreground">{group.description}</p>
        </div>
        <div className="flex shrink-0 flex-col items-start gap-1 sm:items-end">
          <span className="text-xs text-muted-foreground">
            {group.updated_at ? `上次保存于 ${formatDateTime(group.updated_at)}` : '尚未保存过, 使用默认值'}
          </span>
          <Button type="button" variant="ghost" size="sm" onClick={() => setResetting(true)} disabled={pending}>
            <RotateCcw />
            恢复默认值
          </Button>
        </div>
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

      <ResetSettingDialog
        group={group}
        open={resetting}
        dirty={dirty}
        onClose={() => setResetting(false)}
        onReset={applySaved}
      />
    </section>
  );
}
