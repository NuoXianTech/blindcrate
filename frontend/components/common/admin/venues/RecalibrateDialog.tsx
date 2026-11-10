'use client';

import {useState} from 'react';
import {CircleCheck, CircleX, LoaderCircle} from 'lucide-react';
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
import type {RecalibrateData} from '@/lib/services/venue';

/**
 * 重新推导全部地图: 确认后按当前的玩法平衡设置逐个推导, 完成后在对话框中列出每个地图的结果
 */
export function RecalibrateDialog({open, onClose, onSaved}: {
  open: boolean;
  onClose: () => void;
  onSaved: () => void;
}) {
  const {pending, submit} = useSubmit();
  const [data, setData] = useState<RecalibrateData | null>(null);

  const close = () => {
    setData(null);
    onClose();
  };

  const handleConfirm = submit(async () => {
    const result = await services.venueAdmin.recalibrateVenues();
    const updated = result.results.filter((r) => r.updated).length;
    toast.success(`已按玩法平衡设置 v${result.version} 重新推导 ${updated}/${result.results.length} 个地图`);
    setData(result);
    onSaved();
  });

  return (
    <AlertDialog open={open} onOpenChange={(next) => !next && !pending && close()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>重新推导全部地图</AlertDialogTitle>
          <AlertDialogDescription>
            {data ?
              data.results.every((r) => r.updated) ?
                `全部地图已按玩法平衡设置 v${data.version} 保存。` :
                '未保存的地图保留原来的参数, 调整后再保存, 或先停用再推导。' :
              '按当前的玩法平衡设置逐个重新模拟仓库、推导拍卖参数并验算对局, 每个地图需要几秒。启用的地图验算未通过时不保存, 保留原来的参数; 进行中的对局不受影响。'}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {data && (
          <ul className="flex max-h-80 flex-col gap-2 overflow-y-auto text-sm">
            {data.results.map((r) => (
              <li key={r.id} className="flex gap-2">
                {r.updated ?
                  <CircleCheck className="mt-0.5 size-4 shrink-0 text-quality-green" aria-label="已保存" /> :
                  <CircleX className="mt-0.5 size-4 shrink-0 text-destructive" aria-label="未保存" />}
                <div className="flex min-w-0 flex-col">
                  <span>
                    {r.name} <span className="font-mono text-xs text-muted-foreground">{r.code}</span>
                  </span>
                  {r.reason && <span className="text-xs text-muted-foreground">{r.reason}</span>}
                </div>
              </li>
            ))}
          </ul>
        )}
        <AlertDialogFooter>
          {data ? (
            <AlertDialogCancel>关闭</AlertDialogCancel>
          ) : (
            <>
              <AlertDialogCancel disabled={pending}>取消</AlertDialogCancel>
              <AlertDialogAction disabled={pending} onClick={handleConfirm}>
                {pending && <LoaderCircle className="animate-spin" />}
                开始推导
              </AlertDialogAction>
            </>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
