'use client';

import {useEffect, useRef, useState} from 'react';
import {nanoid} from 'nanoid';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {useCaptchaDialog} from '@/components/common/captcha';
import {QuantityStepper, SubmitButton} from '@/components/common/form';
import {CancelledError, check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import {ApiError} from '@/lib/services/core';
import type {CreateShopOrderRequest, ShopItem} from '@/lib/services/shop';
import {parseAmount} from '@/lib/utils';
import {PriceText, RewardList} from '@/components/common/reward';
import {scaleLines, toBundle, unavailableReason} from './constants';

/**
 * 购买对话框的Props接口
 */
interface BuyDialogProps {
  /** null 时关闭 */
  item: ShopItem | null;
  reading: boolean;
  readError: string;
  onClose: () => void;
  /** 重新读取商品和当前可购买份数, 失败或被更新的请求取代时返回 null */
  onRefresh: () => Promise<ShopItem[] | null>;
}

/**
 * 购买确认: 选择份数, 显示总价和得到的全部商品
 */
export function BuyDialog({item, onClose, ...props}: BuyDialogProps) {
  return (
    <Dialog open={!!item} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>购买 {item?.name}</DialogTitle>
          <DialogDescription>确认后立即扣除价格并发放商品, 购买后不能退货</DialogDescription>
        </DialogHeader>
        {/* key 保证每次打开都是新的表单和幂等键 */}
        {item && <BuyForm key={item.code} item={item} onClose={onClose} {...props} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 购买表单
 */
function BuyForm({item, reading, readError, onClose, onRefresh}: Omit<BuyDialogProps, 'item'> & { item: ShopItem }) {
  const {pending, submit} = useSubmit();
  const captcha = useCaptchaDialog('shop');
  const max = item.max_quantity;
  const ready = !reading && !readError;
  const active = useRef(true);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  const [text, setText] = useState('1');
  // 幂等键在份数变化时重新生成, 份数不变的重复提交(如超时后重试)只成交一次
  const [idempotencyKey, setIdempotencyKey] = useState(() => nanoid());
  // 响应丢失时用原请求确认订单, 即使扣款后的可买数量已变成 0 也允许幂等重试。
  const [retry, setRetry] = useState<{request: CreateShopOrderRequest; item: ShopItem} | null>(null);
  const retryRequest = retry?.request;
  const parsed = parseAmount(text);
  const quantity = parsed !== null && parsed >= 1 && parsed <= max ? parsed : null;
  const preview = retry?.item ?? item;
  const previewQuantity = retryRequest?.quantity ?? quantity ?? 1;

  const changeText = (next: string) => {
    setText(next);
    setIdempotencyKey(nanoid());
  };

  const handleSubmit = (event: React.FormEvent) => submit(async () => {
    if (!retryRequest) {
      check(ready ? null : '请先读取当前可购买数量');
      check(unavailableReason(item));
      check(quantity === null ? `份数需为 1-${max}` : null);
    }
    // 开启人机验证时先弹出验证, 没有完成就关闭弹窗则不购买
    const token = await captcha.verify();
    if (!active.current) throw new CancelledError();
    if (!retryRequest) {
      const current = (await onRefresh())?.find((it) => it.code === item.code);
      if (!active.current) throw new CancelledError();
      check(current ? null : '无法确认当前商品, 请刷新后重试');
      check(unavailableReason(current!));
      check(quantity! > current!.max_quantity ? `当前最多可购买 ${current!.max_quantity} 份, 请重新确认数量` : null);
      check(JSON.stringify(toBundle(current!.price)) !== JSON.stringify(toBundle(item.price)) ? '商品价格已变动, 请重新确认' : null);
    }
    const request = retryRequest ?? {code: item.code, quantity: quantity!, price: toBundle(item.price), idempotency_key: idempotencyKey};
    try {
      const order = await services.shop.buy(request, token);
      toast.success(`已购买 ${order.name} ×${order.quantity}`);
    } catch (err) {
      if (!(err instanceof ApiError) || err.status === undefined || err.status >= 500) setRetry((previous) => previous ?? {request, item});
      // 价格变化等冲突, 刷新商品后让玩家重新确认
      if (err instanceof ApiError && err.status !== undefined && [400, 403, 404, 409].includes(err.status)) await onRefresh();
      throw err;
    }
    void onRefresh();
    onClose();
  })(event);

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm text-muted-foreground">{reading ? '正在读取可购买数量…' : `份数 (最多 ${max})`}</span>
        <QuantityStepper label="购买份数" maxLabel="最多" text={text} max={max} invalid={!retryRequest && quantity === null} onChange={changeText} disabled={pending || !ready || max === 0 || !!retryRequest} />
      </div>

      {retryRequest ? (
        <p role="alert" className="text-sm text-muted-foreground">上次购买结果尚未确认，请重试确认订单，同一订单不会重复扣款。</p>
      ) : readError ? (
        <div className="flex items-center justify-between gap-2 text-sm">
          <p role="alert" className="text-destructive">{readError}</p>
          <Button type="button" variant="outline" size="sm" onClick={() => void onRefresh()} disabled={reading || pending}>重新读取</Button>
        </div>
      ) : !reading && (max === 0 || quantity === null) ? (
        <p role="alert" className="text-sm text-destructive">{max === 0 ? '当前无法购买，请检查余额、支付物品和限购条件。' : `当前最多可购买 ${max} 份，请调整数量。`}</p>
      ) : null}

      <div className="flex flex-col gap-3 rounded-lg border bg-card p-4 text-sm">
        <div className="flex flex-col gap-2">
          <span className="text-muted-foreground">得到</span>
          <RewardList lines={scaleLines(preview.goods, previewQuantity)} />
        </div>
        <div className="flex items-start justify-between gap-4 border-t pt-3 font-bold">
          <span>总价</span>
          <PriceText lines={scaleLines(preview.price, previewQuantity)} className="justify-end" />
        </div>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          取消
        </Button>
        <SubmitButton pending={pending} disabled={!retryRequest && (!ready || quantity === null)}>
          {retryRequest ? '重试确认订单' : '确认购买'}
        </SubmitButton>
      </DialogFooter>

      {captcha.dialog}
    </form>
  );
}
