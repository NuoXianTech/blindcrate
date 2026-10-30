'use client';

import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {RewardList} from '@/components/common/reward';
import type {ClaimMissionResult} from '@/lib/services/mission';

/**
 * 领取任务奖励的结果: 得到的货币和物品
 */
export function ClaimResultDialog({result, onClose}: { result: ClaimMissionResult | null; onClose: () => void }) {
  return (
    <Dialog open={!!result} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>领取成功</DialogTitle>
          <DialogDescription>奖励已放进背包和钱包</DialogDescription>
        </DialogHeader>
        {result && <RewardList lines={result.received} />}
        <DialogFooter>
          <Button onClick={onClose}>好的</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
