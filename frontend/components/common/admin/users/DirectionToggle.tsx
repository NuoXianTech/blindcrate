import {Minus, Plus} from 'lucide-react';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import type {Direction} from './constants';

/**
 * 加款/扣款、发放/扣除的切换
 */
export function DirectionToggle({
  value,
  onChange,
  labels,
  disabled,
}: {
  value: Direction;
  onChange: (value: Direction) => void;
  labels: [string, string];
  disabled: boolean;
}) {
  return (
    <div className="grid grid-cols-2 gap-1 rounded-lg bg-muted p-1">
      {(['add', 'sub'] as const).map((d, i) => (
        <Button
          key={d}
          type="button"
          size="sm"
          variant={value === d ? 'secondary' : 'ghost'}
          className={cn(value === d && 'bg-background shadow-xs hover:bg-background', value === d && d === 'sub' && 'text-destructive')}
          onClick={() => onChange(d)}
          disabled={disabled}
        >
          {d === 'add' ? <Plus /> : <Minus />}
          {labels[i]}
        </Button>
      ))}
    </div>
  );
}
