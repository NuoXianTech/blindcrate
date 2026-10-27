"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Eraser, Globe, LoaderCircle, Send, TriangleAlert, Undo2 } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useAuth } from "@/components/common/auth/AuthProvider";
import { SubmitButton } from "@/components/common/form/SubmitButton";
import { check, useSubmit } from "@/components/common/form/use-submit";
import { AdminService, type SettingField, type SettingGroup, type UpdateSettingRequest } from "@/lib/services/admin";
import { toErrorMessage } from "@/lib/services/core";
import { formatDateTime } from "@/lib/format";
import { refreshPublicSettings } from "@/lib/site-actions";

/** 表单里的值: 开关为布尔, 其余都按字符串编辑 */
type FormValues = Record<string, string | boolean>;

function toFormValues(group: SettingGroup): FormValues {
  const values: FormValues = {};
  for (const f of group.fields) {
    const v = group.value[f.key];
    values[f.key] = f.type === "bool" ? !!v : f.secret ? "" : String(v ?? "");
  }
  return values;
}

/** 与已保存的值比较, 只返回改动的字段; 密钥字段填了就算改动, 整数字段保留原始输入, 提交时再校验 */
function diff(group: SettingGroup, values: FormValues): FormValues {
  const changes: FormValues = {};
  for (const f of group.fields) {
    const v = values[f.key];
    if (typeof v === "boolean") {
      if (v !== !!group.value[f.key]) changes[f.key] = v;
    } else if (f.secret) {
      if (v !== "") changes[f.key] = v;
    } else if (v.trim() !== String(group.value[f.key] ?? "")) {
      changes[f.key] = v.trim();
    }
  }
  return changes;
}

/** 系统设置: 每个设置组一张卡片, 表单由后端返回的字段描述生成 */
export function SettingsManager() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.is_admin;
  const [groups, setGroups] = useState<SettingGroup[] | null>(null);

  useEffect(() => {
    // 确认是管理员后再拉取, 否则 AdminGuard 会跳走
    if (!isAdmin) return;
    AdminService.listSettings()
      .then((data) => setGroups(data.groups))
      .catch((err) => toast.error(toErrorMessage(err)));
  }, [isAdmin]);

  const replaceGroup = (updated: SettingGroup) => {
    setGroups((prev) => prev?.map((g) => (g.key === updated.key ? updated : g)) ?? null);
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold">系统设置</h1>
        <p className="text-sm text-muted-foreground">保存后立即生效</p>
      </div>
      {groups ? (
        groups.map((g) => <SettingGroupCard key={g.key} group={g} onSaved={replaceGroup} />)
      ) : (
        <div className="flex h-48 items-center justify-center rounded-xl border bg-card">
          <LoaderCircle className="size-5 animate-spin text-primary" />
        </div>
      )}
    </div>
  );
}

function SettingGroupCard({ group, onSaved }: { group: SettingGroup; onSaved: (group: SettingGroup) => void }) {
  const router = useRouter();
  const { pending, submit } = useSubmit();
  const [values, setValues] = useState(() => toFormValues(group));
  // 要清空的密钥字段
  const [clearing, setClearing] = useState<string[]>([]);
  const changes = diff(group, values);
  const dirty = Object.keys(changes).length > 0 || clearing.length > 0;

  const reset = (g: SettingGroup) => {
    setValues(toFormValues(g));
    setClearing([]);
  };

  const setValue = (key: string, value: string | boolean) => setValues((prev) => ({ ...prev, [key]: value }));

  const toggleClear = (key: string) => {
    setValue(key, "");
    setClearing((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
  };

  const handleSubmit = submit(async () => {
    const body: UpdateSettingRequest = { value: {}, clear_secrets: clearing };
    for (const f of group.fields) {
      if (!(f.key in changes)) continue;
      const v = changes[f.key];
      if (f.type === "int") {
        check(/^-?\d+$/.test(v as string) ? null : `${f.label}需为整数`);
        body.value[f.key] = Number(v);
      } else {
        body.value[f.key] = v;
      }
    }
    const updated = await AdminService.updateSetting(group.key, body);
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
          {group.updated_at ? `上次保存于 ${formatDateTime(group.updated_at)}` : "尚未保存过, 使用默认值"}
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

      {group.key === "mail" && <MailTest dirty={dirty} />}
    </section>
  );
}

function SettingInput({
  field: f,
  id,
  value,
  onChange,
  disabled,
  secretSet,
  secretBroken,
  clearing,
  onToggleClear,
}: {
  field: SettingField;
  id: string;
  value: string | boolean;
  onChange: (value: string | boolean) => void;
  disabled: boolean;
  secretSet: boolean;
  secretBroken: boolean;
  clearing: boolean;
  onToggleClear: () => void;
}) {
  const label = (
    <Label htmlFor={id} className="gap-1.5">
      {f.label}
      {f.public && (
        <Badge variant="outline" className="px-1.5 py-0 text-[10px] font-normal text-muted-foreground" title="会展示给所有访客">
          <Globe />
          公开
        </Badge>
      )}
    </Label>
  );
  const help = f.help && <span className="text-xs text-muted-foreground">{f.help}</span>;

  if (f.input === "switch") {
    return (
      <div className="flex items-center justify-between gap-4 rounded-lg border p-3 sm:col-span-2">
        <div className="flex flex-col gap-1">
          {label}
          {help}
        </div>
        <Switch id={id} checked={value as boolean} onCheckedChange={onChange} disabled={disabled} />
      </div>
    );
  }

  if (f.input === "textarea") {
    return (
      <div className="flex flex-col gap-2 sm:col-span-2">
        {label}
        <Textarea id={id} value={value as string} onChange={(e) => onChange(e.target.value)} disabled={disabled} />
        {help}
      </div>
    );
  }

  if (f.secret) {
    return (
      <div className="flex flex-col gap-2">
        {label}
        <div className="flex gap-2">
          <Input
            id={id}
            type="password"
            autoComplete="new-password"
            placeholder={clearing ? "保存后清空" : secretSet ? "已设置, 留空表示不修改" : "未设置"}
            value={value as string}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled || clearing}
          />
          {(secretSet || secretBroken) && (
            <Button
              type="button"
              variant={clearing ? "secondary" : "outline"}
              size="icon"
              title={clearing ? "取消清空" : "清空"}
              aria-label={clearing ? "取消清空" : "清空"}
              onClick={onToggleClear}
              disabled={disabled}
            >
              <Eraser />
            </Button>
          )}
        </div>
        {secretBroken && !clearing ? (
          <span className="flex items-center gap-1 text-xs text-destructive">
            <TriangleAlert className="size-3.5" />
            无法解密, 请重新填写或清空后再保存
          </span>
        ) : (
          help
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {label}
      <Input
        id={id}
        inputMode={f.input === "number" ? "numeric" : undefined}
        value={value as string}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      />
      {help}
    </div>
  );
}

function MailTest({ dirty }: { dirty: boolean }) {
  const { user } = useAuth();
  const { pending, submit } = useSubmit();
  const [to, setTo] = useState(user?.email ?? "");

  const handleSubmit = submit(async () => {
    check(to.trim() ? null : "请填写收件人");
    await AdminService.testMail(to.trim());
    toast.success(`测试邮件已发送到 ${to.trim()}`);
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-2 border-t pt-6" noValidate>
      <Label htmlFor="mail-test-to">发送测试邮件</Label>
      <div className="flex gap-2">
        <Input
          id="mail-test-to"
          type="email"
          placeholder="收件人邮箱"
          value={to}
          onChange={(e) => setTo(e.target.value)}
          disabled={pending}
          className="sm:max-w-xs"
        />
        <SubmitButton pending={pending} variant="secondary">
          {!pending && <Send />}
          发送
        </SubmitButton>
      </div>
      <span className={cn("text-xs", dirty ? "text-quality-gold" : "text-muted-foreground")}>
        {dirty ? "使用已保存的设置发送, 当前未保存的修改不会生效" : "使用已保存的设置发送, 未启用邮件发送时也可以测试"}
      </span>
    </form>
  );
}
