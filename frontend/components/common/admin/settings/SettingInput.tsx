import {Eraser, Globe, TriangleAlert} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Switch} from '@/components/ui/switch';
import {Textarea} from '@/components/ui/textarea';
import type {SettingField} from '@/lib/services/admin';

/**
 * 设置项输入框的Props接口
 */
interface SettingInputProps {
  /** 后端返回的字段描述 */
  field: SettingField;
  id: string;
  value: string | boolean;
  onChange: (value: string | boolean) => void;
  disabled: boolean;
  /** 密钥字段是否已设置 */
  secretSet: boolean;
  /** 密钥字段是否无法解密 */
  secretBroken: boolean;
  /** 密钥字段是否标记为保存后清空 */
  clearing: boolean;
  onToggleClear: () => void;
}

/**
 * 按字段描述渲染开关、多行文本、密钥或普通输入框
 */
export function SettingInput({
  field: f,
  id,
  value,
  onChange,
  disabled,
  secretSet,
  secretBroken,
  clearing,
  onToggleClear,
}: SettingInputProps) {
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

  if (f.input === 'switch') {
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

  if (f.input === 'textarea') {
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
            placeholder={clearing ? '保存后清空' : secretSet ? '已设置, 留空表示不修改' : '未设置'}
            value={value as string}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled || clearing}
          />
          {(secretSet || secretBroken) && (
            <Button
              type="button"
              variant={clearing ? 'secondary' : 'outline'}
              size="icon"
              title={clearing ? '取消清空' : '清空'}
              aria-label={clearing ? '取消清空' : '清空'}
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
        inputMode={f.input === 'number' ? 'numeric' : undefined}
        value={value as string}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      />
      {help}
    </div>
  );
}
