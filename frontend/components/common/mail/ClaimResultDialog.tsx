'use client';

import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {RewardList} from '@/components/common/reward';
import type {ClaimMailsResult} from '@/lib/services/mailbox';

/**
 * 领取附件的结果: 得到的货币和物品, 一键领取时说明跳过的封数
 */
export function ClaimResultDialog({result, onClose}: { result: ClaimMailsResult | null; onClose: () => void }) {
  return (
    <Dialog open={!!result} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>领取成功</DialogTitle>
          <DialogDescription>
            {result && result.claimed > 1 ? `领取了 ${result.claimed} 封邮件的附件` : '附件已放进背包和钱包'}
            {result?.skipped ? `, ${result.skipped} 封暂时无法领取, 请稍后再试` : ''}
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
