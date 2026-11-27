'use client';

import {createContext, useCallback, useContext, useEffect, useMemo, useRef, useState} from 'react';
import {useAuth} from '@/components/common/auth';
import {useWsConnected, useWsEvent} from '@/hooks/use-ws';
import services from '@/lib/services';
import {ApiError, toErrorMessage} from '@/lib/services/core';
import {PLAYER_SETTINGS_CHANGED, type PlayerSettings} from '@/lib/services/player-settings';

interface Value {
  settings: PlayerSettings | null;
  error: string;
  saving: boolean;
  refresh: () => Promise<PlayerSettings>;
  save: (keepOne: boolean) => Promise<void>;
  /** 游戏设置弹窗是否打开, 弹窗见 GameSettingsDialog */
  dialogOpen: boolean;
  openDialog: () => void;
  closeDialog: () => void;
}

const Context = createContext<Value | null>(null);

/** 每个账号使用独立状态, 切换账号时丢弃请求和偏好。 */
export function PlayerSettingsProvider({children}: {children: React.ReactNode}) {
  const {user} = useAuth();
  return <Session key={user?.id ?? 0} userID={user?.id}>{children}</Session>;
}

function Session({userID, children}: {userID?: number; children: React.ReactNode}) {
  const [settings, setSettings] = useState<PlayerSettings | null>(null);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const latest = useRef<PlayerSettings | null>(null);
  const alive = useRef(true);
  const savingRef = useRef(false);
  const readID = useRef(0);
  const connected = useWsConnected();

  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);

  const accept = useCallback((next: PlayerSettings) => {
    if (alive.current && (!latest.current || next.version >= latest.current.version)) {
      latest.current = next;
      setSettings(next);
      setError('');
    }
  }, []);

  const refresh = useCallback(async () => {
    if (!userID) throw new ApiError('请先登录');
    const id = ++readID.current;
    const startedVersion = latest.current?.version;
    try {
      const next = await services.playerSettings.read();
      if (!alive.current) throw new ApiError('账号已切换, 请重新操作');
      accept(next);
      return latest.current ?? next;
    } catch (err) {
      // 旧请求失败不能覆盖后续请求或推送已经读到的设置。
      if (alive.current && id === readID.current && latest.current?.version === startedVersion) setError(toErrorMessage(err));
      throw err;
    }
  }, [userID, accept]);

  const save = useCallback(async (keepOne: boolean) => {
    if (savingRef.current || !latest.current || !userID) return;
    savingRef.current = true;
    setSaving(true);
    try {
      const next = await services.playerSettings.update({keep_one_collectible: keepOne, version: latest.current.version});
      if (!alive.current) throw new ApiError('账号已切换, 请重新操作');
      accept(next);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409 && alive.current) await refresh().catch(() => undefined);
      throw err;
    } finally {
      savingRef.current = false;
      if (alive.current) setSaving(false);
    }
  }, [userID, accept, refresh]);

  useEffect(() => {
    if (userID) void refresh().catch(() => undefined);
  }, [userID, connected, refresh]);

  useWsEvent<PlayerSettings>(PLAYER_SETTINGS_CHANGED, (next) => {
    if (userID) accept(next);
  });

  useEffect(() => {
    const visible = () => {
      if (userID && document.visibilityState === 'visible') void refresh().catch(() => undefined);
    };
    document.addEventListener('visibilitychange', visible);
    return () => document.removeEventListener('visibilitychange', visible);
  }, [userID, refresh]);

  const openDialog = useCallback(() => setDialogOpen(true), []);
  const closeDialog = useCallback(() => setDialogOpen(false), []);

  const value = useMemo(
      () => ({settings, error, saving, refresh, save, dialogOpen, openDialog, closeDialog}),
      [settings, error, saving, refresh, save, dialogOpen, openDialog, closeDialog],
  );
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function usePlayerSettings() {
  const value = useContext(Context);
  if (!value) throw new Error('usePlayerSettings requires PlayerSettingsProvider');
  return value;
}
