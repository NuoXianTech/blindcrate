'use client';

import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import type {RewardLine} from '@/lib/services/reward';
import {RewardList} from './Lines';

/**
 * 领取结果对话框的Props接口
 */
interface RewardResultDialogProps {
  /** 得到的货币和物品, null 时关闭 */
  lines: RewardLine[] | null;
  title?: string;
  description?: React.ReactNode;
  onClose: () => void;
}

/**
 * 领取奖励的结果: 得到的货币和物品, 任务、签到、邮件附件等共用
 */
export function RewardResultDialog({lines, title = '领取成功', description = '奖励已放进背包和钱包', onClose}: RewardResultDialogProps) {
  return (
    <Dialog open={!!lines} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        {lines && <RewardList lines={lines} />}
        <DialogFooter>
          <Button onClick={onClose}>好的</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
