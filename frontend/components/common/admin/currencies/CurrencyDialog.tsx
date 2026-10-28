'use client';

import {useState} from 'react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Switch} from '@/components/ui/switch';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {Currency, CurrencyRequest} from '@/lib/services/admin';
import {formatAmount, parseAmount} from '@/lib/utils';
import {CODE_PATTERN, MAX_DECIMALS, MAX_SORT_ORDER} from './constants';

/** 预览小数位时使用的金额, 按最小单位 */
const SAMPLE_AMOUNT = 12345;

/**
 * 货币对话框的Props接口
 */
interface CurrencyDialogProps {
  /** null: 关闭; "new": 新建; 货币: 编辑 */
  currency: Currency | 'new' | null;
  /** 是否为基础货币, 基础货币不能停用 */
  isBase: boolean;
  /** 新建时默认填写的排序, 排在现有货币之后 */
  nextSortOrder: number;
  onClose: () => void;
  /** 保存成功后刷新列表 */
  onSaved: () => void;
}

/**
 * 新建或编辑货币: 代码创建后不能修改
 */
export function CurrencyDialog({currency, ...props}: CurrencyDialogProps) {
  const editing = currency && currency !== 'new' ? currency : null;
  return (
    <Dialog open={!!currency} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? `编辑 ${editing.name}` : '新建货币'}</DialogTitle>
          <DialogDescription>新货币可以用于物品回收价、地图出价和掉落池; 停用后不能再入账或扣款, 玩家已有的余额保留</DialogDescription>
        </DialogHeader>
        {/* key 保证切换货币时表单重置 */}
        {currency && <CurrencyForm key={editing?.code ?? 'new'} currency={editing} {...props} />}
      </DialogContent>
    </Dialog>
  );
}

/** 输入框的原文换算为整数, 格式不对或超出范围时中断提交 */
function readInteger(text: string, label: string, min: number, max: number): number {
  const value = parseAmount(text);
  check(value === null || value < min || value > max ? `${label}需为 ${min}-${max}` : null);
  return value as number;
}

/**
 * 货币表单, currency 为 null 时新建
 */
function CurrencyForm({currency, isBase, nextSortOrder, onClose, onSaved}: Omit<CurrencyDialogProps, 'currency'> & {
  currency: Currency | null;
}) {
  const {pending, submit} = useSubmit();
  const [code, setCode] = useState(currency?.code ?? '');
  const [name, setName] = useState(currency?.name ?? '');
  const [decimals, setDecimals] = useState(String(currency?.decimals ?? 0));
  const [sortOrder, setSortOrder] = useState(String(currency?.sort_order ?? nextSortOrder));
  const [enabled, setEnabled] = useState(currency?.is_enabled ?? true);

  const handleSubmit = submit(async () => {
    const body: CurrencyRequest = {
      name: name.trim(),
      decimals: readInteger(decimals, '小数位', 0, MAX_DECIMALS),
      sort_order: readInteger(sortOrder, '排序', -MAX_SORT_ORDER, MAX_SORT_ORDER),
      is_enabled: enabled,
    };
    check(body.name.length === 0 || [...body.name].length > 32 ? '名称需为 1-32 个字符' : null);
    if (currency) {
      await services.admin.updateCurrency(currency.code, body);
      toast.success(`已保存 ${body.name}`);
    } else {
      const created = {...body, code: code.trim()};
      check(CODE_PATTERN.test(created.code) ? null : '代码需为 1-32 个小写字母、数字或下划线');
      await services.admin.createCurrency(created);
      toast.success(`已新建 ${body.name}`);
    }
    onSaved();
    onClose();
  });

  const previewDecimals = parseAmount(decimals);

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="currency-code">代码</Label>
          <Input
            id="currency-code"
            maxLength={32}
            placeholder="如 gem"
            className="font-mono"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            disabled={pending || !!currency}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="currency-name">名称</Label>
          <Input id="currency-name" maxLength={32} value={name} onChange={(e) => setName(e.target.value)} disabled={pending} />
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">余额、流水和配置都按代码引用货币, 代码创建后不能修改</span>

      <div className="grid grid-cols-2 gap-3">
        <div className="flex flex-col gap-2">
          <Label htmlFor="currency-decimals">小数位</Label>
          <Input
            id="currency-decimals"
            inputMode="numeric"
            className="font-mono tabular-nums"
            value={decimals}
            onChange={(e) => setDecimals(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="currency-sort-order">排序</Label>
          <Input
            id="currency-sort-order"
            inputMode="numeric"
            className="font-mono tabular-nums"
            value={sortOrder}
            onChange={(e) => setSortOrder(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>
      <span className="-mt-2 text-xs text-muted-foreground">
        小数位为 0-{MAX_DECIMALS}
        {previewDecimals !== null && previewDecimals >= 0 && previewDecimals <= MAX_DECIMALS &&
          `, 最小单位的 ${SAMPLE_AMOUNT} 显示为 ${formatAmount(SAMPLE_AMOUNT, previewDecimals)}`}
        ; 已被余额、物品、地图或掉落池使用的货币不能修改小数位; 排序从小到大决定余额和下拉框中的顺序
      </span>

      <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
        <div className="flex flex-col gap-0.5">
          <Label htmlFor="currency-enabled">启用</Label>
          <span className="text-xs text-muted-foreground">
            {isBase ? '基础货币不能停用, 请先在系统设置中更换基础货币' : '停用后使用它的物品不能卖出, 地图不能开局'}
          </span>
        </div>
        <Switch
          id="currency-enabled"
          checked={enabled}
          onCheckedChange={setEnabled}
          disabled={pending || (isBase && enabled)}
        />
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending}>{currency ? '保存' : '新建'}</SubmitButton>
      </DialogFooter>
    </form>
  );
}
