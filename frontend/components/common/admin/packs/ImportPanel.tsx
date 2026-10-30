'use client';

import {useRef, useState} from 'react';
import {FileArchive, LoaderCircle, RefreshCw, Upload} from 'lucide-react';
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
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Switch} from '@/components/ui/switch';
import {useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {PackAction, PackActionType, PackEntryResult, PackImport, PackPreview, PackStatus} from '@/lib/services/admin';
import {ApiError} from '@/lib/services/core';
import {formatDateTime} from '@/lib/utils';
import {PackEntriesTable} from './PackEntriesTable';
import {ACTION_LABELS, MAX_PACK_SIZE, STATUS_LABELS, entryID, formatSize} from './constants';
import {sectionLabel, usePackSections} from './use-pack-sections';

/** 一项配置选择的处理方式 */
interface Choice {
  action: PackActionType;
  code: string;
}

/** 没有选择时的处理方式 */
const DEFAULT_CHOICE: Choice = {action: 'overwrite', code: ''};

/**
 * 导入配置包: 上传后预览每一项与已有配置的比较结果, 逐项选择导入、跳过或改名导入, 确认后在一个事务中导入
 */
export function ImportPanel({onImported}: { onImported: (record: PackImport) => void }) {
  const inputRef = useRef<HTMLInputElement>(null);
  const {sections} = usePackSections();
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<PackPreview | null>(null);
  const [choices, setChoices] = useState<Record<string, Choice>>({});
  // 处理方式修改后还没有重新预览
  const [dirty, setDirty] = useState(false);
  const [showSame, setShowSame] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const previewing = useSubmit();
  const importing = useSubmit();
  const pending = previewing.pending || importing.pending;

  const actionsOf = (next: Record<string, Choice>): PackAction[] =>
    (preview?.entries ?? [])
        .filter((e) => next[entryID(e)] && next[entryID(e)].action !== 'overwrite')
        .map((e) => ({section: e.section, key: e.key, action: next[entryID(e)].action, code: next[entryID(e)].code.trim()}));

  const runPreview = (target: File, next: Record<string, Choice>) => previewing.submit(async () => {
    const result = await services.admin.previewPack(target, actionsOf(next));
    setPreview(result);
    setDirty(false);
  });

  const handleFile = (selected: File | undefined) => {
    // 清空以便再次选择同一个文件
    if (inputRef.current) inputRef.current.value = '';
    if (!selected) return;
    if (selected.size > MAX_PACK_SIZE) {
      toast.error(`配置包不能超过 ${MAX_PACK_SIZE >> 20} MB`);
      return;
    }
    setFile(selected);
    setPreview(null);
    setChoices({});
    void runPreview(selected, {})();
  };

  const choose = (e: PackEntryResult, choice: Partial<Choice>) => {
    setChoices((prev) => {
      const id = entryID(e);
      return {...prev, [id]: {...(prev[id] ?? DEFAULT_CHOICE), ...choice}};
    });
    setDirty(true);
  };

  const handleImport = importing.submit(async () => {
    if (!file) return;
    try {
      const record = await services.admin.importPack(file, actionsOf(choices));
      toast.success(`已导入 ${record.name}: 新增 ${record.created} 项, 覆盖 ${record.updated} 项, 跳过 ${record.skipped} 项`);
      setConfirming(false);
      setFile(null);
      setPreview(null);
      setChoices({});
      onImported(record);
    } catch (err) {
      // 有冲突时 data 为每一项的结果, 换成最新的结果以便逐项处理
      if (err instanceof ApiError && err.status === 409 && Array.isArray(err.data)) {
        setPreview((prev) => prev && {...prev, entries: err.data as PackEntryResult[]});
        setDirty(false);
      }
      setConfirming(false);
      throw err;
    }
  });

  const entries = preview?.entries ?? [];
  const counts = entries.reduce<Record<PackStatus, number>>(
      (acc, e) => ({...acc, [e.status]: acc[e.status] + 1}),
      {new: 0, changed: 0, same: 0, conflict: 0},
  );
  // 冲突项需要跳过或改名后才能导入, 改名后是否还有冲突以重新预览的结果为准
  const unresolved = entries.filter((e) => e.status === 'conflict' && (choices[entryID(e)]?.action ?? 'overwrite') === 'overwrite').length;
  const visible = showSame ? entries : entries.filter((e) => e.status !== 'same' || choices[entryID(e)]);

  const renderAction = (e: PackEntryResult) => {
    const choice = choices[entryID(e)] ?? DEFAULT_CHOICE;
    const renamable = sections.some((s) => s.code === e.section && s.renamable);
    return (
      <div className="flex items-center gap-2">
        <Select value={choice.action} onValueChange={(action) => choose(e, {action: action as PackActionType})} disabled={pending}>
          <SelectTrigger size="sm" className="w-28" aria-label="处理方式">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {(Object.keys(ACTION_LABELS) as PackActionType[])
                .filter((a) => a !== 'rename' || renamable)
                .map((a) => (
                  <SelectItem key={a} value={a}>
                    {ACTION_LABELS[a]}
                  </SelectItem>
                ))}
          </SelectContent>
        </Select>
        {choice.action === 'rename' && (
          <Input
            aria-label="新代码"
            placeholder="新代码"
            className="h-8 w-36 font-mono"
            maxLength={64}
            value={choice.code}
            onChange={(ev) => choose(e, {code: ev.target.value})}
            disabled={pending}
          />
        )}
      </div>
    );
  };

  const manifest = preview?.manifest;
  const fields = manifest ? [
    {label: '名称', value: manifest.name},
    {label: '作者', value: manifest.author || '-'},
    {label: '导出时间', value: formatDateTime(manifest.exported_at)},
    {label: '格式版本', value: String(manifest.version)},
    {label: '大小', value: formatSize(preview.size)},
    {label: '配置种类', value: manifest.sections.map((s) => sectionLabel(sections, s)).join('、')},
  ] : [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 rounded-xl border bg-card p-4">
        <div className="flex flex-wrap items-center gap-3">
          <input
            ref={inputRef}
            type="file"
            accept=".zip,application/zip"
            className="hidden"
            onChange={(e) => handleFile(e.target.files?.[0])}
          />
          <Button type="button" variant="outline" onClick={() => inputRef.current?.click()} disabled={pending}>
            {previewing.pending ? <LoaderCircle className="animate-spin" /> : <Upload />}
            选择配置包
          </Button>
          {file && (
            <span className="flex items-center gap-1.5 text-sm">
              <FileArchive className="size-4 text-muted-foreground" />
              {file.name}
            </span>
          )}
        </div>
        <span className="text-xs text-muted-foreground">
          导入只新增或覆盖配置, 不会停用或删除已有配置, 也不改动玩家数据; 物品按 ID 对应, 品质按等级, 设置按设置组, 其他配置按代码。
          导入的设置会保存为新版本, 之后可以在系统设置中回滚
        </span>
      </div>

      {preview && (
        <>
          <dl className="grid gap-x-6 gap-y-2 rounded-xl border bg-card p-4 text-sm sm:grid-cols-3">
            {fields.map((f) => (
              <div key={f.label} className="flex flex-col gap-0.5">
                <dt className="text-xs text-muted-foreground">{f.label}</dt>
                <dd>{f.value}</dd>
              </div>
            ))}
            <div className="flex flex-col gap-0.5 sm:col-span-3">
              <dt className="text-xs text-muted-foreground">SHA-256</dt>
              <dd className="font-mono text-xs break-all">{preview.digest}</dd>
            </div>
          </dl>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
            <span className="text-sm">
              {(Object.keys(STATUS_LABELS) as PackStatus[]).map((s) => `${STATUS_LABELS[s].label} ${counts[s]}`).join(' · ')}
            </span>
            <div className="flex items-center gap-2">
              <Switch id="pack-show-same" checked={showSame} onCheckedChange={setShowSame} />
              <Label htmlFor="pack-show-same">显示相同的配置</Label>
            </div>
            <div className="ml-auto flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={file ? runPreview(file, choices) : undefined}
                disabled={pending || !dirty}
                title="按选择的处理方式重新比较, 改名导入的结果以重新预览为准"
              >
                {previewing.pending ? <LoaderCircle className="animate-spin" /> : <RefreshCw />}
                重新预览
              </Button>
              <Button type="button" onClick={() => setConfirming(true)} disabled={pending}>
                导入
              </Button>
            </div>
          </div>
          {unresolved > 0 && (
            <span className="text-sm text-destructive">
              有 {unresolved} 项冲突, 请跳过或改名导入后再导入
            </span>
          )}

          <PackEntriesTable
            entries={visible}
            renderAction={renderAction}
            empty={counts.same ? '与已有配置全部相同, 打开"显示相同的配置"查看' : '配置包中没有配置'}
          />
        </>
      )}

      <AlertDialog open={confirming} onOpenChange={(open) => !open && !importing.pending && setConfirming(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>导入配置包 {manifest?.name}</AlertDialogTitle>
            <AlertDialogDescription>
              按选择的处理方式在一个事务中导入, 有任何一项无法导入时全部不写入; 覆盖的配置立即生效, 无法撤销, 设置可以在系统设置中回滚。
              {dirty && ' 处理方式修改后还没有重新预览, 导入的结果以实际导入为准。'}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={importing.pending}>取消</AlertDialogCancel>
            <AlertDialogAction disabled={importing.pending} onClick={handleImport}>
              {importing.pending && <LoaderCircle className="animate-spin" />}
              导入
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
