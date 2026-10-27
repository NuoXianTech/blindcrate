'use client';

import {useEffect} from 'react';
import {useRouter} from 'next/navigation';
import {LoaderCircle} from 'lucide-react';
import {Button} from '@/components/ui/button';
import {useAuth} from '@/components/common/auth';
import {ROUTES, loginHref} from '@/lib/access';

/** 开始游戏: 已登录直接跳转游戏首页, 未登录点击后先去登录 */
export function StartGameButton() {
  const router = useRouter();
  const {user, loading} = useAuth();

  useEffect(() => {
    if (user) router.replace(ROUTES.gameHome);
  }, [user, router]);

  const busy = loading || !!user;

  return (
    <Button size="lg" className="glow" disabled={busy} onClick={() => router.push(loginHref(ROUTES.gameHome))}>
      {busy && <LoaderCircle className="animate-spin" />}
      开始游戏
    </Button>
  );
}
