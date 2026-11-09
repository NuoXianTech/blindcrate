'use client';

import {useState} from 'react';
import {useForm} from 'react-hook-form';
import {Pencil} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Form, FormControl, FormField, FormItem, FormMessage} from '@/components/ui/form';
import {Input} from '@/components/ui/input';
import {Popover, PopoverContent, PopoverTrigger} from '@/components/ui/popover';
import {SubmitButton} from '@/components/common/form';
import {CurrencyIcon} from '@/components/common/wallet';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {Venue} from '@/lib/services/venue';
import {amountInput, formatAmount, parseAmount} from '@/lib/utils';

interface CarryValues {
  amount: string;
}

/** 快捷比例, 按带入资金上限 */
const CARRY_SHARES = [0.25, 0.5, 0.75, 1];

/**
 * 带入金额: 整个拍场共用这笔钱, 在 [起拍价, 带入资金上限] 内自选, 准入时按它冻结; 没选过时为上限
 * 带得少不容易被套住, 但后面的拍品可能出不起价
 */
export function CarryPicker({venue: v, carry, decimals, disabled, onSaved}: {
  venue: Venue;
  /** 当前选的带入金额, 已按上限截断 */
  carry: number;
  decimals: number;
  disabled: boolean;
  onSaved: (amount: number) => void;
}) {
  const [open, setOpen] = useState(false);
  const {pending, submit} = useSubmit();
  const {start_price: min, max_carry: max} = v.auction;
  const form = useForm<CarryValues>({defaultValues: {amount: amountInput(carry, decimals)}});

  const handleSubmit = form.handleSubmit(({amount}) => submit(async () => {
    const saved = await services.auction.setCarry({venue: v.code, amount: parseAmount(amount, decimals) ?? 0});
    onSaved(saved.amount);
    setOpen(false);
  })());

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        if (next) form.reset({amount: amountInput(carry, decimals)});
        setOpen(next);
      }}
    >
      <PopoverTrigger asChild>
        <Button variant="ghost" size="xs" disabled={disabled} className="h-auto gap-1 px-1 font-mono tabular-nums">
          <CurrencyIcon currency={v.bid_currency} className="size-4" />
          {formatAmount(carry, decimals)}
          <Pencil className="size-3 text-muted-foreground" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72">
        <Form {...form}>
          <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
            <div className="flex flex-col gap-1">
              <span className="text-sm font-medium">带入金额</span>
              <span className="text-xs text-muted-foreground">
                整场的拍品共用这笔钱, 开局时从可用余额冻结, 结束后解冻剩下的; 可选 {formatAmount(min, decimals)} - {formatAmount(max, decimals)}
              </span>
            </div>
            <FormField
              control={form.control}
              name="amount"
              rules={{
                validate: (value) => {
                  const amount = parseAmount(value, decimals);
                  if (amount === null) return '请输入金额';
                  if (amount < min || amount > max) return `需在 ${formatAmount(min, decimals)} - ${formatAmount(max, decimals)} 之间`;
                  return true;
                },
              }}
              render={({field}) => (
                <FormItem>
                  <FormControl>
                    <Input inputMode="decimal" aria-label="带入金额" className="font-mono tabular-nums" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex flex-wrap gap-1.5">
              {CARRY_SHARES.map((share) => (
                <Button
                  key={share}
                  type="button"
                  size="xs"
                  variant="outline"
                  onClick={() => form.setValue('amount', amountInput(Math.max(min, Math.floor(max * share)), decimals), {shouldValidate: true})}
                >
                  {share === 1 ? '上限' : `${share * 100}%`}
                </Button>
              ))}
            </div>
            <SubmitButton pending={pending}>保存</SubmitButton>
          </form>
        </Form>
      </PopoverContent>
    </Popover>
  );
}
