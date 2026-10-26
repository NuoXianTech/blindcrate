"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { LoaderCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/components/common/auth/AuthProvider";

const GAME_HOME_PATH = "/home";

/** 开始游戏: 已登录直接跳转游戏首页, 未登录点击后先去登录 */
export function StartGameButton() {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (user) router.replace(GAME_HOME_PATH);
  }, [user, router]);

  const busy = loading || !!user;

  return (
    <Button
      size="lg"
      className="glow"
      disabled={busy}
      onClick={() => router.push(`/login?redirect=${encodeURIComponent(GAME_HOME_PATH)}`)}
    >
      {busy && <LoaderCircle className="animate-spin" />}
      开始游戏
    </Button>
  );
}
