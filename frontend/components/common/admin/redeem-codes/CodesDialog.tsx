'use client';

import {useEffect, useState} from 'react';
import {Copy, Download} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Label} from '@/components/ui/label';
import {Skeleton} from '@/components/ui/skeleton';
import {Switch} from '@/components/ui/switch';
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from '@/components/ui/table';
import {Pagination, TableEmpty} from '@/components/common/layout';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {RedeemBatch, RedeemCode} from '@/lib/services/redeem';
import {CODE_PAGE_SIZE, codeAvailable, usesText} from './constants';

/** 复制到剪贴板, 浏览器不允许时提示手动复制 */
async function copyText(text: string, done: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success(done);
  } catch {
    toast.error('复制失败, 请手动选中后复制');
  }
}

/** 保存为文本文件, 每行一个兑换码 */
function downloadCodes(batch: RedeemBatch, codes: RedeemCode[]) {
  const blob = new Blob([codes.map((c) => c.code).join('\r\n') + '\r\n'], {type: 'text/plain;charset=utf-8'});
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  // 批次名称中不能用作文件名的字符换成下划线
  link.download = `兑换码-${batch.name.replace(/[\\/:*?"<>|\s]+/g, '_')}-${batch.id}.txt`;
  link.click();
  // 立即释放时部分浏览器会取消下载
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

/**
 * 查看批次的兑换码: 复制、下载, 以及每个兑换码已兑换的次数
 */
export function CodesDialog({batch, onClose}: {
  /** null 时关闭 */
  batch: RedeemBatch | null;
  onClose: () => void;
}) {
  return (
    <Dialog open={!!batch} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{batch?.name} 的兑换码</DialogTitle>
          <DialogDescription>
            {batch && `共 ${batch.code_count} 个, ${usesText(batch.max_uses)}; 玩家输入时不区分大小写, 可以带空格和连字符`}
          </DialogDescription>
        </DialogHeader>
        {/* key 保证切换批次时重新拉取 */}
        {batch && <CodeList key={batch.id} batch={batch} />}
      </DialogContent>
    </Dialog>
  );
}

/**
 * 兑换码列表, 打开时一次拉取批次的全部兑换码, 在页面上分页和筛选
 */
function CodeList({batch}: { batch: RedeemBatch }) {
  const [codes, setCodes] = useState<RedeemCode[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [onlyAvailable, setOnlyAvailable] = useState(false);
  const [current, setCurrent] = useState(1);

  useEffect(() => {
    let alive = true;
    services.redeemAdmin.listCodes(batch.id)
        .then((data) => {
          if (alive) setCodes(data.codes);
        })
        .catch((err) => {
          toast.error(toErrorMessage(err));
          if (alive) setFailed(true);
        });
    return () => {
      alive = false;
    };
  }, [batch.id]);

  if (failed) return <p className="py-6 text-center text-sm text-muted-foreground">兑换码加载失败, 请关闭后重试</p>;
  if (!codes) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-9" />
        <Skeleton className="h-9" />
        <Skeleton className="h-9" />
      </div>
    );
  }

  const available = codes.filter((c) => codeAvailable(c.used, batch.max_uses));
  const shown = onlyAvailable ? available : codes;
  const used = codes.reduce((sum, c) => sum + c.used, 0);
  const pages = Math.max(1, Math.ceil(shown.length / CODE_PAGE_SIZE));
  // 筛选后页数变少时回到最后一页
  const page = Math.min(current, pages);
  const rows = shown.slice((page - 1) * CODE_PAGE_SIZE, page * CODE_PAGE_SIZE);
  const scope = onlyAvailable ? '还能兑换的' : '全部';

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-col gap-1.5">
          <span className="text-sm text-muted-foreground">
            已兑换 {used} 次, 还能兑换的有 {available.length} 个
          </span>
          <Label className="font-normal">
            <Switch
              checked={onlyAvailable}
              onCheckedChange={(checked) => {
                setOnlyAvailable(checked);
                setCurrent(1);
              }}
            />
            只看还能兑换的
          </Label>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            disabled={!shown.length}
            onClick={() => copyText(shown.map((c) => c.code).join('\n'), `已复制${scope} ${shown.length} 个兑换码`)}
          >
            <Copy />
            复制{scope}
          </Button>
          <Button variant="outline" disabled={!shown.length} onClick={() => downloadCodes(batch, shown)}>
            <Download />
            下载 TXT
          </Button>
        </div>
      </div>

      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>兑换码</TableHead>
              <TableHead className="text-right">已兑换</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.length ? (
              rows.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-mono select-all">{c.code}</TableCell>
                  <TableCell className="text-right font-mono tabular-nums">
                    {c.used}
                    {batch.max_uses > 0 && <span className="text-muted-foreground"> / {batch.max_uses}</span>}
                  </TableCell>
                  <TableCell>
                    <Button variant="ghost" size="icon-sm" title="复制" aria-label={`复制 ${c.code}`} onClick={() => copyText(c.code, `已复制 ${c.code}`)}>
                      <Copy />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableEmpty colSpan={3}>{onlyAvailable ? '兑换码都已兑换完' : '这个批次没有兑换码'}</TableEmpty>
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination current={page} size={CODE_PAGE_SIZE} total={shown.length} onChange={setCurrent} />
    </div>
  );
}
