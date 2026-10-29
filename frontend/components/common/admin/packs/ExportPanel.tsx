'use client';

import {useState} from 'react';
import {Download} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {Switch} from '@/components/ui/switch';
import {SubmitButton} from '@/components/common/form';
import {check, useSubmit} from '@/hooks/use-submit';
import services from '@/lib/services';
import type {PackSection} from '@/lib/services/admin';
import {MAX_NAME_LENGTH, SECTIONS, SECTION_HINTS, SECTION_LABELS, formatSize} from './constants';

/**
 * 导出配置包: 填写名称和作者, 选择配置种类后下载 zip
 */
export function ExportPanel() {
  const [name, setName] = useState('');
  const [author, setAuthor] = useState('');
  const [sections, setSections] = useState<PackSection[]>(SECTIONS);
  const {pending, submit} = useSubmit();

  const toggle = (section: PackSection, on: boolean) => {
    // 保持依赖顺序
    setSections((prev) => SECTIONS.filter((s) => (s === section ? on : prev.includes(s))));
  };

  const handleSubmit = submit(async () => {
    check(name.trim() ? null : '请填写配置包名称');
    check(sections.length ? null : '请至少选择一种配置');
    const {blob, filename} = await services.admin.exportPack({name: name.trim(), author: author.trim(), sections});
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.click();
    // 立即释放时部分浏览器会取消下载
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    toast.success(`已导出 ${filename} (${formatSize(blob.size)})`);
  });

  return (
    <form className="flex max-w-2xl flex-col gap-4 rounded-xl border bg-card p-4" onSubmit={handleSubmit}>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-2">
          <Label htmlFor="pack-name">名称</Label>
          <Input
            id="pack-name"
            maxLength={MAX_NAME_LENGTH}
            placeholder="如 正式服 10 月配置"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={pending}
          />
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="pack-author">作者</Label>
          <Input
            id="pack-author"
            maxLength={MAX_NAME_LENGTH}
            placeholder="可选"
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
            disabled={pending}
          />
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <Label>配置种类</Label>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => setSections(sections.length === SECTIONS.length ? [] : SECTIONS)}
            disabled={pending}
          >
            {sections.length === SECTIONS.length ? '全不选' : '全选'}
          </Button>
        </div>
        <div className="grid gap-2 sm:grid-cols-2">
          {SECTIONS.map((s) => (
            <div key={s} className="flex items-center justify-between gap-4 rounded-lg border p-3">
              <div className="flex flex-col gap-0.5">
                <Label htmlFor={`pack-section-${s}`}>{SECTION_LABELS[s]}</Label>
                <span className="text-xs text-muted-foreground">{SECTION_HINTS[s]}</span>
              </div>
              <Switch
                id={`pack-section-${s}`}
                checked={sections.includes(s)}
                onCheckedChange={(on) => toggle(s, on)}
                disabled={pending}
              />
            </div>
          ))}
        </div>
        <span className="text-xs text-muted-foreground">
          已删除的配置不导出; 引用了其他配置时(如掉落池中的物品), 导入的环境中需要有对应的配置, 或一起导出
        </span>
      </div>

      <SubmitButton pending={pending} className="self-start">
        {!pending && <Download />}
        导出
      </SubmitButton>
    </form>
  );
}
