import {Minus, Plus} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {parseAmount} from '@/lib/utils';

/**
 * 数量步进器: 减、输入框、加, 以及一键填入上限
 */
export function QuantityStepper({label, maxLabel = '全部', text, max, invalid, onChange, disabled}: {
  /** 输入框的无障碍名称, 如 "出售数量" */
  label: string;
  /** 填入上限的按钮文字 */
  maxLabel?: string;
  text: string;
  max: number;
  invalid: boolean;
  onChange: (text: string) => void;
  disabled?: boolean;
}) {
  const value = parseAmount(text) ?? 0;
  return (
    <div className="flex shrink-0 items-center gap-1">
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-8"
        aria-label="减少"
        onClick={() => onChange(String(Math.min(max, Math.max(1, value - 1))))}
        disabled={disabled || value <= 1}
      >
        <Minus />
      </Button>
      <Input
        inputMode="numeric"
        aria-label={label}
        aria-invalid={invalid}
        className="h-8 w-16 text-center font-mono tabular-nums"
        value={text}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      />
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-8"
        aria-label="增加"
        onClick={() => onChange(String(Math.min(max, Math.max(1, value + 1))))}
        disabled={disabled || value >= max}
      >
        <Plus />
      </Button>
      <Button type="button" variant="ghost" size="sm" className="h-8 px-2" onClick={() => onChange(String(max))} disabled={disabled || value === max}>
        {maxLabel}
      </Button>
    </div>
  );
}
