'use client';

import {cn} from 'cn';
import {Badge} from '@/components/ui/badge';
import {Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {ItemImage, SellPriceText} from '@/components/common/item';
import {useQualities} from '@/components/common/site';
import type {CodexEntry, CodexSet} from '@/lib/services/codex';
import type {BalanceItem} from '@/lib/services/wallet';
import {formatDateTime} from '@/lib/utils';
import {SILHOUETTE, entryName} from './constants';

/**
 * 藏品详情对话框的Props接口
 */
interface CodexEntryDialogProps {
  /** 要查看的藏品, null 时关闭 */
  entry: CodexEntry | null;
  /** 全部收集组, 用于列出收录这件藏品的收集组 */
  sets: CodexSet[];
  /** 货币列表, 用于展示售价的货币名称和小数位 */
  currencies: BalanceItem[];
  onClose: () => void;
}

/**
 * 藏品详情: 收集过的显示描述、当前的出售价格、背包中的数量和第一次获得的时间;
 * 没有收集的名称显示为问号, 只显示品质; 都列出收录它的收集组
 */
export function CodexEntryDialog({entry, sets, currencies, onClose}: CodexEntryDialogProps) {
  const q = useQualities().qualityOf(entry?.quality ?? 0);
  const included = entry ? sets.filter((s) => s.item_ids.includes(entry.item_id)) : [];

  return (
    <Dialog open={!!entry} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        {entry && (
          <>
            <DialogHeader className="flex-row items-center gap-4 text-left">
              <span className="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-card">
                <ItemImage item={entry} className={cn(!entry.collected && SILHOUETTE)} />
              </span>
              <div className="flex min-w-0 flex-col gap-1">
                <DialogTitle className={cn('truncate', entry.collected ? q.text : 'text-muted-foreground')}>{entryName(entry)}</DialogTitle>
                <DialogDescription className="flex flex-wrap items-center gap-2">
                  <span>{q.name}品质</span>
                  <Badge variant={entry.collected ? 'secondary' : 'outline'}>{entry.collected ? '已收集' : '未收集'}</Badge>
                </DialogDescription>
              </div>
            </DialogHeader>

            {entry.collected ? (
              <>
                {entry.description && <p className="text-sm whitespace-pre-wrap text-muted-foreground">{entry.description}</p>}
                <dl className="grid grid-cols-2 gap-3 rounded-lg border bg-card p-4 text-sm">
                  <Detail label="出售价格">
                    <SellPriceText item={entry} currencies={currencies} />
                  </Detail>
                  <Detail label="背包中">
                    <span className="font-mono tabular-nums">{entry.quantity}</span>
                  </Detail>
                  <Detail label="第一次获得">{formatDateTime(entry.first_at)}</Detail>
                </dl>
              </>
            ) : (
              <p className="text-sm text-muted-foreground">还没有获得过这件藏品, 获得后会收进图鉴并显示详情, 之后卖出或用掉也会保留。</p>
            )}

            {included.length > 0 && (
              <div className="flex flex-col gap-1.5">
                <span className="text-xs text-muted-foreground">收录于</span>
                <div className="flex flex-wrap gap-1.5">
                  {included.map((s) => (
                    <Badge key={s.code} variant="outline">
                      {s.name}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 详情里的一项
 */
function Detail({label, children}: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}
