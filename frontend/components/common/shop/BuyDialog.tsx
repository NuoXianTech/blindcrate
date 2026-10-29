'use client';

import {useState} from 'react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {QuantityStepper, SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {ApiError} from '@/lib/services/core';
import type {ShopItem} from '@/lib/services/shop';
import {parseAmount} from '@/lib/utils';
import {GoodsList, PriceText} from './Lines';
import {maxQuantity, scaleLines, toBundle} from './constants';

/**
 * 购买对话框的Props接口
 */
interface BuyDialogProps {
  /** null 时关闭 */
  item: ShopItem | null;
  onClose: () => void;
  /** 购买成功, 或商品已变化需要刷新时调用 */
  onRefresh: () => void;
}

/**
 * 购买确认: 选择份数, 显示总价和得到的全部商品
 */
export function BuyDialog({item, onClose, onRefresh}: BuyDialogProps) {
  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>购买 {item?.name}</DialogTitle>
          <DialogDescription>确认后立即扣除价格并发放商品, 购买后不能退货</DialogDescription>
        </DialogHeader>
        {/* key 保证每次打开都是新的表单和幂等键 */}
        {item && <BuyForm key={item.code} item={item} onClose={onClose} onRefresh={onRefresh} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 购买表单
 */
function BuyForm({item, onClose, onRefresh}: Omit<BuyDialogProps, 'item'> & { item: ShopItem }) {
  const {pending, submit} = useSubmit();
  const max = maxQuantity(item);
  const [text, setText] = useState('1');
  // 幂等键在份数变化时重新生成, 份数不变的重复提交(如超时后重试)只成交一次
  const [idempotencyKey, setIdempotencyKey] = useState(() => nanoid());
  const parsed = parseAmount(text);
  const quantity = parsed !== null && parsed >= 1 && parsed <= max ? parsed : null;

  const changeText = (next: string) => {
    setText(next);
    setIdempotencyKey(nanoid());
  };

  const handleSubmit = submit(async () => {
    check(quantity === null ? `份数需为 1-${max}` : null);
    try {
      await services.shop.buy({code: item.code, quantity: quantity!, price: toBundle(item.price), idempotency_key: idempotencyKey});
    } catch (err) {
      // 价格变化等冲突, 刷新商品后让玩家重新确认
      if (err instanceof ApiError && err.status === 409) onRefresh();
      throw err;
    }
    toast.success(`已购买 ${item.name} ×${quantity}`);
    onRefresh();
    onClose();
  });

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm text-muted-foreground">份数 (最多 {max})</span>
        <QuantityStepper label="购买份数" maxLabel="最多" text={text} max={max} invalid={quantity === null} onChange={changeText} disabled={pending} />
      </div>

      <div className="flex flex-col gap-3 rounded-lg border bg-card p-4 text-sm">
        <div className="flex flex-col gap-2">
          <span className="text-muted-foreground">得到</span>
          <GoodsList lines={scaleLines(item.goods, quantity ?? 1)} />
        </div>
        <div className="flex items-start justify-between gap-4 border-t pt-3 font-bold">
          <span>总价</span>
          <PriceText lines={scaleLines(item.price, quantity ?? 1)} className="justify-end" />
        </div>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending} disabled={quantity === null}>
          确认购买
        </SubmitButton>
      </DialogFooter>
    </form>
  );
}
