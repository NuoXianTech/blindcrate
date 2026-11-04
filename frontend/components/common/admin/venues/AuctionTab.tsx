'use client';

import {Sparkles} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {formatRate} from '@/lib/auction';
import type {Currency} from '@/lib/services/admin';
import type {AuctionRule} from '@/lib/services/venue';
import {formatAmount} from '@/lib/utils';
import {FieldGroup, InputField} from './Fields';
import {recommendedQuickWin, resizeQuickWin, type FormTabProps} from './form';

/** 验算要求的轮数, 与后端 venue 的闸门一致 */
const MIN_ROUNDS = 4;

/**
 * 拍卖: 轮数和时长、速胜倍率和加轮; 金额和结算比例由后端按模拟的价值分布推导, 这里只展示
 */
export function AuctionTab({form, onChange, disabled, currency, derived, stale}: FormTabProps & {
  /** 出价货币, 金额按它的小数位展示 */
  currency?: Currency;
  /** 最近一次模拟或保存时推导出的参数, 还没有时为空 */
  derived: AuctionRule | null;
  /** 推导之后表单有改动 */
  stale: boolean;
}) {
  const changeRounds = (rounds: string) => {
    const value = Number(rounds);
    // 轮数合法时速胜倍率跟着增减, 否则先保留, 提交时会提示轮数不对
    const valid = Number.isInteger(value) && value >= 1 && value <= 10;
    onChange(valid ? {rounds, quickWin: resizeQuickWin(form.quickWin, value)} : {rounds});
  };
  const rounds = Number(form.rounds);
  const roundsValid = Number.isInteger(rounds) && rounds >= 1 && rounds <= 10;

  return (
    <div className="flex flex-col gap-6">
      <FieldGroup title="轮次">
        <div className="grid gap-3 sm:grid-cols-3">
          <InputField
            id="venue-rounds"
            label="轮数"
            help={`1-10 轮; 验算要求至少 ${MIN_ROUNDS} 轮, 否则不能启用`}
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

      <FieldGroup
        title="速胜倍率"
        help="前几轮结束时, 第一名的出价不低于第二名的这么多倍、且不低于速胜下限就直接成交, 1-10 倍; 最后一轮不需要"
      >
        {form.quickWin.length > 0 ? (
          <>
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
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="self-start"
              onClick={() => onChange({quickWin: recommendedQuickWin(rounds)})}
              disabled={disabled || !roundsValid}
            >
              <Sparkles />
              使用推荐倍率
            </Button>
          </>
        ) : (
          <p className="text-sm text-muted-foreground">只有一轮, 不需要速胜倍率</p>
        )}
      </FieldGroup>

      <DerivedRule derived={derived} currency={currency} stale={stale} />
    </div>
  );
}

/**
 * 推导出的金额和结算比例, 只读
 */
function DerivedRule({derived, currency, stale}: { derived: AuctionRule | null; currency?: Currency; stale: boolean }) {
  const amount = (value: number) => `${formatAmount(value, currency?.decimals ?? 0)} ${currency?.name ?? ''}`.trim();
  // 旧数据中推导的字段都是 0
  const ready = derived && derived.start_price > 0;
  const rows = ready ?
    [
      {label: '入场费', value: amount(derived.entry_fee), help: '开局时与入场代价一起扣除, 练习局不扣'},
      {label: '起拍价', value: amount(derived.start_price), help: '出价不能低于它'},
      {label: '带入资金上限', value: amount(derived.max_carry), help: '开局时冻结 min(可用余额, 上限)'},
      {
        label: '人机带入资金',
        value: amount(derived.bot_carry || derived.max_carry),
        help: '人机的出价不超过它; 人机的钱不来自真人, 价值远高于平均的仓库留给真人去争',
      },
      {label: '速胜下限', value: amount(derived.speed_floor), help: '速胜时第一名的出价不能低于它'},
      {
        label: '领先折让',
        value: formatRate(derived.lead_discount),
        help: '成交价 = 第二名的出价 + 领先部分 × (1 - 折让), 让成交价平均落在仓库价值扣掉出售手续费',
      },
      {label: '奖池', value: `入场费的 ${formatRate(derived.pool_rate)}`, help: '成交后分给出过价的真人输家, 名次越前分得越多'},
      {
        label: '超付进奖池',
        value: `${formatRate(derived.rebate_rate)}, 每局最多 ${amount(derived.rebate_cap)}`,
        help: '真人成交价超过仓库价值时, 超出部分按比例进奖池',
      },
      {
        label: '压轴揭示',
        value: derived.finale ? '开启' : '关闭',
        help: '开启时最后一轮公开品质的是仓库里最贵的一件, 否则公开顺序全部随机',
      },
    ] :
    [];

  return (
    <FieldGroup
      title="金额和结算"
      help="按排位模式模拟的仓库价值分布和玩法平衡设置自动推导, 模拟或保存时更新; 换了藏品、掉落或玩法平衡设置后重新保存即可, 不需要手动调"
    >
      {ready ? (
        <div className="flex flex-col gap-2 rounded-lg border p-3 text-sm">
          {stale && <p className="text-xs text-muted-foreground">配置已修改, 重新模拟后更新</p>}
          {rows.map((row) => (
            <div key={row.label} className="grid grid-cols-[7rem_1fr] gap-x-3 gap-y-0.5">
              <span className="text-muted-foreground">{row.label}</span>
              <span className="font-mono tabular-nums">{row.value}</span>
              <span />
              <span className="text-xs text-muted-foreground">{row.help}</span>
            </div>
          ))}
        </div>
      ) : (
        <p className="rounded-lg border p-3 text-sm text-muted-foreground">
          还没有推导, 点击下方的模拟或保存后生成; 旧地图在重新保存前按旧规则进行
        </p>
      )}
    </FieldGroup>
  );
}
