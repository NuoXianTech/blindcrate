import {Eraser, Globe, Plus, Trash2, TriangleAlert} from 'lucide-react';
import {Badge} from '@/components/ui/badge';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Switch} from '@/components/ui/switch';
import {Textarea} from '@/components/ui/textarea';
import type {SettingField} from '@/lib/services/admin';
import {type FormValue, UNIT_SUFFIX} from './constants';

/**
 * 设置项输入框的Props接口
 */
interface SettingInputProps {
  /** 后端返回的字段描述 */
  field: SettingField;
  id: string;
  value: FormValue;
  onChange: (value: FormValue) => void;
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
 * 按字段描述渲染开关、多行文本、密钥、下拉框、整数列表或普通输入框
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

  if (f.input === 'select') {
    const known = f.options?.some((o) => o.value === value);
    return (
      <div className="flex flex-col gap-2">
        {label}
        <Select value={value as string} onValueChange={onChange} disabled={disabled}>
          <SelectTrigger id={id} className="w-full">
            <SelectValue placeholder="请选择" />
          </SelectTrigger>
          <SelectContent>
            {!known && value !== '' && <SelectItem value={value as string}>不可选的值 {value as string}</SelectItem>}
            {f.options?.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {help}
      </div>
    );
  }

  if (f.input === 'list') {
    const items = value as string[];
    const update = (index: number, item: string) => onChange(items.map((v, i) => (i === index ? item : v)));
    return (
      <div className="flex flex-col gap-2 sm:col-span-2">
        {label}
        <div className="flex flex-wrap items-center gap-2">
          {items.map((item, i) => (
            <div key={i} className="flex items-center gap-1">
              <NumberInput
                id={i === 0 ? id : undefined}
                aria-label={`${f.label}第 ${i + 1} 项`}
                unit={f.unit}
                className="w-28"
                value={item}
                onChange={(v) => update(i, v)}
                disabled={disabled}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                title="删除"
                aria-label="删除"
                className="hover:text-destructive"
                onClick={() => onChange(items.filter((_, j) => j !== i))}
                disabled={disabled}
              >
                <Trash2 />
              </Button>
            </div>
          ))}
          <Button type="button" variant="outline" size="sm" onClick={() => onChange([...items, ''])} disabled={disabled}>
            <Plus />
            添加
          </Button>
        </div>
        {help}
      </div>
    );
  }

  if (f.input === 'number') {
    return (
      <div className="flex flex-col gap-2">
        {label}
        <NumberInput id={id} unit={f.unit} value={value as string} onChange={onChange} disabled={disabled} />
        {help}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {label}
      <Input id={id} value={value as string} onChange={(e) => onChange(e.target.value)} disabled={disabled} />
      {help}
    </div>
  );
}

/**
 * 整数输入框的Props接口
 */
interface NumberInputProps {
  id?: string;
  'aria-label'?: string;
  unit?: SettingField['unit'];
  className?: string;
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
}

/**
 * 整数输入框, 有单位时在右侧显示; 基点按百分比输入
 */
function NumberInput({id, unit, className, value, onChange, disabled, ...props}: NumberInputProps) {
  const suffix = unit && UNIT_SUFFIX[unit];
  return (
    <div className={`relative ${className ?? ''}`}>
      <Input
        id={id}
        aria-label={props['aria-label']}
        inputMode={unit === 'bp' ? 'decimal' : 'numeric'}
        className={`font-mono tabular-nums ${suffix ? (suffix.length > 1 ? 'pr-12' : 'pr-7') : ''}`}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      />
      {suffix && (
        <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm text-muted-foreground">
          {suffix}
        </span>
      )}
    </div>
  );
}
