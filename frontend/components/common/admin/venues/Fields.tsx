import {cn} from 'cn';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Switch} from '@/components/ui/switch';

/**
 * 输入框字段的Props接口
 */
interface InputFieldProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  /** 输入框下方的说明 */
  help?: string;
  /** 输入框内右侧的单位, 如 % */
  suffix?: string;
  /** 数值输入时为 numeric 或 decimal, 文本输入时为空 */
  inputMode?: 'numeric' | 'decimal';
  placeholder?: string;
  maxLength?: number;
  disabled?: boolean;
}

/**
 * 带标签、单位和说明的输入框, 数值输入使用等宽数字
 */
export function InputField({id, label, value, onChange, help, suffix, inputMode, placeholder, maxLength, disabled}: InputFieldProps) {
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="relative">
        <Input
          id={id}
          inputMode={inputMode}
          placeholder={placeholder}
          maxLength={maxLength}
          className={cn(inputMode && 'font-mono tabular-nums', suffix && 'pr-12')}
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
      {help && <span className="text-xs text-muted-foreground">{help}</span>}
    </div>
  );
}

/**
 * 带说明的开关, 占一整行
 */
export function SwitchField({id, label, help, checked, onChange, disabled}: {
  id: string;
  label: string;
  help?: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
      <div className="flex flex-col gap-0.5">
        <Label htmlFor={id}>{label}</Label>
        {help && <span className="text-xs text-muted-foreground">{help}</span>}
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onChange} disabled={disabled} />
    </div>
  );
}

/**
 * 表单中的一组字段, 带小标题
 */
export function FieldGroup({title, help, children}: { title: string; help?: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-col gap-0.5">
        <h3 className="text-sm font-medium">{title}</h3>
        {help && <p className="text-xs text-muted-foreground">{help}</p>}
      </div>
      {children}
    </section>
  );
}
