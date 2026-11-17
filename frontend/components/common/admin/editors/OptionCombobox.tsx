'use client';

import {useState} from 'react';
import {Check, ChevronsUpDown} from 'lucide-react';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList} from '@/components/ui/command';
import {Popover, PopoverContent, PopoverTrigger} from '@/components/ui/popover';
import {useQualities} from '@/components/common/site';
import type {ConditionOption} from '@/lib/services/condition';

/**
 * 可选值下拉框的Props接口
 */
interface OptionComboboxProps {
  id?: string;
  /** 可以选的可选值, 已经过滤掉不能新选的 */
  options: ConditionOption[];
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  disabled?: boolean;
}

/**
 * 可按名称或说明搜索的下拉框, 可选值较多(如物品)时使用; 带品质的可选值按品质色展示
 */
export function OptionCombobox({id, options, value, onChange, placeholder, disabled}: OptionComboboxProps) {
  const [open, setOpen] = useState(false);
  const {qualityOf} = useQualities();
  const selected = options.find((o) => o.value === value);

  return (
    // modal: 在 Dialog 里也能滚动列表
    <Popover open={open} onOpenChange={setOpen} modal>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between font-normal"
          disabled={disabled}
        >
          {selected ? (
            <span className={cn('truncate', selected.quality && qualityOf(selected.quality).text)}>{selected.label}</span>
          ) : (
            <span className="truncate text-muted-foreground">{value ? `未知选项 ${value}` : placeholder}</span>
          )}
          <ChevronsUpDown className="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-(--radix-popover-trigger-width) p-0" align="start">
        <Command>
          <CommandInput placeholder="搜索" />
          <CommandList>
            <CommandEmpty>没有找到</CommandEmpty>
            <CommandGroup>
              {options.map((o) => (
                <CommandItem
                  key={o.value}
                  value={o.value}
                  keywords={[o.label, o.hint ?? '']}
                  onSelect={() => {
                    onChange(o.value);
                    setOpen(false);
                  }}
                >
                  <span className={cn('truncate', o.quality && qualityOf(o.quality).text)}>{o.label}</span>
                  {o.hint && <span className="text-xs text-muted-foreground">{o.hint}</span>}
                  <Check className={cn('ml-auto', o.value === value ? 'opacity-100' : 'opacity-0')} />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
