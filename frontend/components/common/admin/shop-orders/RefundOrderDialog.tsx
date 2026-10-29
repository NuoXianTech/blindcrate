'use client';

import {LoaderCircle} from 'lucide-react';
import {toast} from 'sonner';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {AdminShopOrder} from '@/lib/services/admin';

/**
 * 订单退款的二次确认
 */
export function RefundOrderDialog({order, price, goods, onClose, onSaved}: {
  order: AdminShopOrder | null;
  /** 价格和商品的文字, 用于确认 */
  price: string;
  goods: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();

  const handleConfirm = submit(async () => {
    if (!order) return;
    await services.admin.refundShopOrder(order.id);
    toast.success(`已退款订单 #${order.id}`);
    onSaved();
    onClose();
  });

  return (
    <AlertDialog open={!!order} onOpenChange={(open) => !open && !pending && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            退款订单 #{order?.id} {order?.name} ×{order?.quantity}
          </AlertDialogTitle>
          <AlertDialogDescription>
            退还 {price || '0'}, 收回 {goods}; 价格中已下架的物品按回收价折成货币退还。用户的货币或物品不足以收回时不能退款。恢复库存, 不恢复限购数量; 无法撤销。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={pending} onClick={handleConfirm}>
            {pending && <LoaderCircle className="animate-spin" />}
            退款
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
