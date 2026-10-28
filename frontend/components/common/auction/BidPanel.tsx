'use client';

import {useState} from 'react';
import {Hand, LoaderCircle} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {CurrencyIcon} from '@/components/common/wallet';
import {formatRatio} from '@/lib/auction';
import services from '@/lib/services';
import type {AuctionView} from '@/lib/services/auction';
import {toErrorMessage} from '@/lib/services/core';
import {amountInput, formatAmount, parseAmount} from '@/lib/utils';

/** 加价按钮, 在当前输入的基础上加这么多比例 */
const RAISE_STEPS = [0.1, 0.25, 0.5];

/**
 * 出价面板: 暗标出价, 不低于起拍价和自己之前的出价, 不超过带入资金; 0 表示本轮不加价
 * 提交后等待其他人, 所有真人都提交或倒计时结束时本轮判定
 */
export function BidPanel({view, decimals, onBid}: {
  view: AuctionView;
  decimals: number;
  /** 出价成功后返回的最新视角 */
  onBid: (view: AuctionView) => void;
}) {
  const {venue, self} = view;
  const min = Math.max(venue.start_price, self.bid);
  const [input, setInput] = useState(() => amountInput(min, decimals));
  const [submitting, setSubmitting] = useState(false);

  const amount = parseAmount(input, decimals);
  const invalid = amount === null || amount < min || amount > self.carry;
  const extra = view.round > venue.rounds;

  const submit = async (value: number) => {
    setSubmitting(true);
    try {
      onBid(await services.auction.bid({match_id: view.match_id, round: view.round, amount: value}));
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setSubmitting(false);
    }
  };

  const raise = (step: number) => {
    const base = amount !== null && amount >= min ? amount : min;
    setInput(amountInput(Math.min(self.carry, Math.ceil(base * (1 + step))), decimals));
  };

  let rule: string;
  if (extra) {
    rule = '加轮: 只有出价并列最高的玩家可以出价, 最高价唯一时成交';
  } else if (view.round < venue.rounds) {
    rule = `第一名的出价达到第二名的 ${formatRatio(venue.quick_win[view.round - 1])} 倍时直接成交`;
  } else {
    rule = '最后一轮: 最高价成交, 并列时加轮';
  }

  if (!self.can_bid) {
    return (
      <div className="flex flex-col items-center gap-1 rounded-xl border bg-card p-4 text-center text-sm">
        <span className="font-medium">{self.submitted ? '已提交本轮出价' : '本轮你不能出价'}</span>
        <span className="text-muted-foreground">
          {self.submitted ? '等待其他玩家出价, 倒计时结束或所有人提交后公布结果' : '等待并列最高的玩家加轮'}
        </span>
      </div>
    );
  }

  return (
    <form
      className="flex flex-col gap-3 rounded-xl border bg-card p-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (!invalid && amount !== null) void submit(amount);
      }}
    >
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">出价</span>
        <span className="flex items-center gap-1 text-muted-foreground">
          带入资金
          <CurrencyIcon currency={venue.bid_currency} className="size-4" />
          <span className="font-mono tabular-nums">{formatAmount(self.carry, decimals)}</span>
        </span>
      </div>

      <div className="flex gap-2">
        <Input
          inputMode="decimal"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          aria-invalid={input !== '' && invalid}
          aria-label="出价金额"
          className="font-mono text-lg tabular-nums"
        />
        <Button type="submit" disabled={invalid || submitting}>
          {submitting && <LoaderCircle className="animate-spin" />}
          出价
        </Button>
      </div>

      <div className="flex flex-wrap gap-1.5">
        <Button type="button" size="xs" variant="outline" onClick={() => setInput(amountInput(min, decimals))}>
          {self.bid ? '保持' : '起拍价'}
        </Button>
        {RAISE_STEPS.map((step) => (
          <Button key={step} type="button" size="xs" variant="outline" onClick={() => raise(step)}>
            +{step * 100}%
          </Button>
        ))}
        <Button type="button" size="xs" variant="outline" onClick={() => setInput(amountInput(self.carry, decimals))}>
          全部
        </Button>
      </div>

      <span className="text-xs text-muted-foreground">
        可出 {formatAmount(min, decimals)} - {formatAmount(self.carry, decimals)}; {rule}
      </span>

      <Button type="button" variant="ghost" size="sm" disabled={submitting} onClick={() => void submit(0)}>
        <Hand />
        {self.bid ? '不加价' : '本轮不出价'}
      </Button>
    </form>
  );
}
