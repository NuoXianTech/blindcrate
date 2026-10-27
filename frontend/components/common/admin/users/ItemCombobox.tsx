'use client';

import {useState} from 'react';
import {Check, ChevronsUpDown} from 'lucide-react';
import {cn} from 'cn';
import {Button} from '@/components/ui/button';
import {Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList} from '@/components/ui/command';
import {Popover, PopoverContent, PopoverTrigger} from '@/components/ui/popover';
import {qualityOf} from '@/lib/quality';

/** 下拉框里的物品 */
export interface ItemOption {
  id: number;
  name: string;
  quality: number;
}

/**
 * 物品下拉框的Props接口
 */
interface ItemComboboxProps {
  options: ItemOption[];
  /** 选中的物品 ID */
  value: number | null;
  onChange: (value: number) => void;
  placeholder: string;
  disabled?: boolean;
}

/**
 * 可按名称或编号搜索的物品下拉框
 */
export function ItemCombobox({options, value, onChange, placeholder, disabled}: ItemComboboxProps) {
  const [open, setOpen] = useState(false);
  const selected = options.find((o) => o.id === value);

  return (
    // modal: 在 Dialog 里也能滚动列表
    <Popover open={open} onOpenChange={setOpen} modal>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between font-normal"
          disabled={disabled}
        >
          {selected ? (
            <span className={cn('truncate', qualityOf(selected.quality).text)}>{selected.name}</span>
          ) : (
            <span className="truncate text-muted-foreground">{placeholder}</span>
          )}
          <ChevronsUpDown className="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-(--radix-popover-trigger-width) p-0" align="start">
        <Command>
          <CommandInput placeholder="搜索名称或编号" />
          <CommandList>
            <CommandEmpty>没有找到物品</CommandEmpty>
            <CommandGroup>
              {options.map((o) => (
                <CommandItem
                  key={o.id}
                  value={String(o.id)}
                  keywords={[o.name, `#${o.id}`]}
                  onSelect={() => {
                    onChange(o.id);
                    setOpen(false);
                  }}
                >
                  <span className={cn('truncate', qualityOf(o.quality).text)}>{o.name}</span>
                  <span className="text-xs text-muted-foreground">#{o.id}</span>
                  <Check className={cn('ml-auto', o.id === value ? 'opacity-100' : 'opacity-0')} />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
