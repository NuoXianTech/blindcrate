'use client';

import {useEffect, useState} from 'react';
import Link from 'next/link';
import {ArrowLeft, BookmarkCheck, LoaderCircle} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {Switch} from '@/components/ui/switch';
import {Label} from '@/components/ui/label';
import {ROUTES} from '@/lib/access';
import {toErrorMessage} from '@/lib/services/core';
import {usePlayerSettings} from './PlayerSettingsProvider';

export function GameSettingsMain() {
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

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold">游戏设置</h1>
        <p className="mt-2 text-sm text-muted-foreground">调整你的游戏习惯。设置自动保存到账号，在其他设备登录后也会生效。</p>
      </div>
      <section className="overflow-hidden rounded-xl border bg-card" aria-labelledby="inventory-settings">
        <div className="border-b px-5 py-4">
          <h2 id="inventory-settings" className="flex items-center gap-2 font-semibold"><BookmarkCheck className="size-4" />背包与出售</h2>
        </div>
        <div className="flex flex-col gap-5 p-5">
          {error ? (
            <div role="alert" className="flex flex-wrap items-center justify-between gap-3 text-sm">
              <span className="text-destructive">游戏设置读取失败：{error}</span>
              <Button variant="outline" onClick={() => void refresh().catch(() => undefined)}>重新读取</Button>
            </div>
          ) : !settings ? (
            <p role="status" className="flex items-center gap-2 text-sm text-muted-foreground"><LoaderCircle className="size-4 animate-spin" />正在读取游戏设置</p>
          ) : (
            <>
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
            </>
          )}
        </div>
      </section>
      <Button variant="outline" className="self-start" asChild><Link href={ROUTES.inventory}><ArrowLeft />返回背包</Link></Button>
    </div>
  );
}
