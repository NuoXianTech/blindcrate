'use client';

import {useEffect, useState} from 'react';
import {LoaderCircle} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle} from '@/components/ui/dialog';
import {Label} from '@/components/ui/label';
import {Switch} from '@/components/ui/switch';
import {toErrorMessage} from '@/lib/services/core';
import {usePlayerSettings} from './PlayerSettingsProvider';

/**
 * 游戏设置弹窗: 放在 PlayerSettingsProvider 内, 各处的入口用 usePlayerSettings().openDialog 打开
 */
export function GameSettingsDialog() {
  const {dialogOpen, closeDialog} = usePlayerSettings();
  return (
    <Dialog open={dialogOpen} onOpenChange={(next) => !next && closeDialog()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>游戏设置</DialogTitle>
          <DialogDescription>调整你的游戏习惯。设置自动保存到账号，在其他设备登录后也会生效。</DialogDescription>
        </DialogHeader>
        {/* 每次打开都重新读取, 保存提示也从头开始 */}
        {dialogOpen && <GameSettings />}
      </DialogContent>
    </Dialog>
  );
}

function GameSettings() {
  const {settings, error, saving, refresh, save} = usePlayerSettings();
  const [saveError, setSaveError] = useState('');
  const [saved, setSaved] = useState(false);
  useEffect(() => {
    void refresh().catch(() => undefined);
  }, [refresh]);

  const change = async (keepOne: boolean) => {
    setSaveError('');
    setSaved(false);
    try {
      await save(keepOne);
      setSaved(true);
    } catch (err) {
      setSaveError(toErrorMessage(err));
    }
  };

  if (error) {
    return (
      <div role="alert" className="flex flex-wrap items-center justify-between gap-3 text-sm">
        <span className="text-destructive">游戏设置读取失败：{error}</span>
        <Button variant="outline" onClick={() => void refresh().catch(() => undefined)}>重新读取</Button>
      </div>
    );
  }
  if (!settings) {
    return <p role="status" className="flex items-center gap-2 text-sm text-muted-foreground"><LoaderCircle className="size-4 animate-spin" />正在读取游戏设置</p>;
  }
  return (
    <section className="flex flex-col gap-4" aria-labelledby="inventory-settings">
      <h3 id="inventory-settings" className="text-sm font-medium text-muted-foreground">背包与出售</h3>
      <div className="flex items-start justify-between gap-6">
        <div className="flex flex-col gap-2">
          <Label htmlFor="keep-one-collectible" className="text-base leading-6">出售时，每种藏品至少保留 1 件</Label>
          <p id="keep-one-help" className="text-sm leading-6 text-muted-foreground">单件和批量出售都会保留每种藏品的最后一件，重复获得的藏品仍可出售。</p>
        </div>
        <Switch id="keep-one-collectible" aria-describedby="keep-one-help" className="mt-1 shrink-0" checked={settings.keep_one_collectible} disabled={saving} onCheckedChange={(next) => void change(next)} />
      </div>
      <div className="border-l-2 border-primary/40 pl-3 text-sm leading-6 text-muted-foreground">
        <p>{settings.keep_one_collectible ? '收藏保护已开启。持有 5 件相同藏品时，最多出售 4 件。' : '收藏保护已关闭，可以出售最后一件藏品。'}</p>
        <p>仅保护出售，不影响其他用途的消耗。道具、箱子、门票及已移除的物品不受此设置影响。</p>
      </div>
      <div role="status" aria-live="polite" className="min-h-5 text-sm">
        {saving ? <span className="text-muted-foreground">正在保存…</span> : saveError ? <span className="text-destructive">{saveError}，可重新切换开关重试。</span> : saved ? <span>已保存</span> : <span className="text-muted-foreground">修改后自动保存</span>}
      </div>
    </section>
  );
}
