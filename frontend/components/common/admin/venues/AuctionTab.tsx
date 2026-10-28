'use client';

import type {Currency} from '@/lib/services/admin';
import {FieldGroup, InputField} from './Fields';
import {resizeQuickWin, type FormTabProps} from './form';

/**
 * 拍卖: 起拍价、带入资金、轮数和时长、速胜倍率和加轮
 */
export function AuctionTab({form, onChange, disabled, currency}: FormTabProps & {
  /** 出价货币, 金额按它的小数位输入 */
  currency?: Currency;
}) {
  const amountMode = currency && currency.decimals > 0 ? 'decimal' : 'numeric';
  const changeRounds = (rounds: string) => {
    const value = Number(rounds);
    // 轮数合法时速胜倍率跟着增减, 否则先保留, 提交时会提示轮数不对
    const valid = Number.isInteger(value) && value >= 1 && value <= 10;
    onChange(valid ? {rounds, quickWin: resizeQuickWin(form.quickWin, value)} : {rounds});
  };

  return (
    <div className="flex flex-col gap-6">
      <FieldGroup title="资金" help="开局时冻结 min(可用余额, 带入资金上限), 出价不能超过冻结的金额">
        <div className="grid gap-3 sm:grid-cols-2">
          <InputField
            id="venue-start-price"
            label="起拍价"
            help="出价不能低于起拍价"
            suffix={currency?.name}
            inputMode={amountMode}
            value={form.startPrice}
            onChange={(startPrice) => onChange({startPrice})}
            disabled={disabled}
          />
          <InputField
            id="venue-max-carry"
            label="带入资金上限"
            help="不能低于起拍价"
            suffix={currency?.name}
            inputMode={amountMode}
            value={form.maxCarry}
            onChange={(maxCarry) => onChange({maxCarry})}
            disabled={disabled}
          />
        </div>
      </FieldGroup>

      <FieldGroup title="轮次">
        <div className="grid gap-3 sm:grid-cols-3">
          <InputField
            id="venue-rounds"
            label="轮数"
            help="1-10 轮"
            inputMode="numeric"
            value={form.rounds}
            onChange={changeRounds}
            disabled={disabled}
          />
          <InputField
            id="venue-round-seconds"
            label="每轮时长"
            help="5-120 秒"
            suffix="秒"
            inputMode="numeric"
            value={form.roundSeconds}
            onChange={(roundSeconds) => onChange({roundSeconds})}
            disabled={disabled}
          />
          <InputField
            id="venue-reveal-seconds"
            label="开箱后停留时长"
            help="全部藏品扫描完后再停留 3-60 秒; 扫描时长在品质管理中按品质配置"
            suffix="秒"
            inputMode="numeric"
            value={form.revealSeconds}
            onChange={(revealSeconds) => onChange({revealSeconds})}
            disabled={disabled}
          />
        </div>
        <InputField
          id="venue-extra-rounds"
          label="加轮上限"
          help="最后一轮最高价并列时加轮, 0-5 轮; 加满后按先出价的人成交"
          inputMode="numeric"
          value={form.maxExtraRounds}
          onChange={(maxExtraRounds) => onChange({maxExtraRounds})}
          disabled={disabled}
        />
      </FieldGroup>

      <FieldGroup title="速胜倍率" help="前几轮结束时, 第一名的出价不低于第二名的这么多倍就直接成交, 1-10 倍; 最后一轮不需要">
        {form.quickWin.length > 0 ? (
          <div className="grid grid-cols-3 gap-3">
            {form.quickWin.map((value, i) => (
              <InputField
                key={i}
                id={`venue-quick-win-${i}`}
                label={`第 ${i + 1} 轮`}
                suffix="倍"
                inputMode="decimal"
                value={value}
                onChange={(x) => onChange({quickWin: form.quickWin.map((y, j) => (j === i ? x : y))})}
                disabled={disabled}
              />
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">只有一轮, 不需要速胜倍率</p>
        )}
      </FieldGroup>
    </div>
  );
}
