'use client';

import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {RewardList} from '@/components/common/reward';
import type {CheckinResult} from '@/lib/services/checkin';

/**
 * 签到的结果: 连续天数、领到第几天的奖励和得到的货币和物品
 */
export function CheckinResultDialog({result, onClose}: { result: CheckinResult | null; onClose: () => void }) {
  return (
    <Dialog open={!!result} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>签到成功</DialogTitle>
          <DialogDescription>
            {result && `已连续签到 ${result.streak} 天, 领取了第 ${result.day} 天的奖励`}
          </DialogDescription>
        </DialogHeader>
        {result && <RewardList lines={result.received} />}
        <DialogFooter>
          <Button onClick={onClose}>好的</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
