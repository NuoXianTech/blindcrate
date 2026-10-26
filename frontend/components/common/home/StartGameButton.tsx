"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { LoaderCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AuthService } from "@/lib/services/auth";

const GAME_HOME_PATH = "/home";

/** 开始游戏: 进入首页时检查登录状态, 已登录直接跳转游戏首页, 未登录点击后先去登录 */
export function StartGameButton() {
  const router = useRouter();
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    AuthService.getUserInfo()
      .then(() => router.replace(GAME_HOME_PATH))
      // 未登录或请求失败都停留在首页, 由用户点击去登录
      .catch(() => setChecking(false));
  }, [router]);

  return (
    <Button
      size="lg"
      className="glow"
      disabled={checking}
      onClick={() => router.push(`/login?redirect=${encodeURIComponent(GAME_HOME_PATH)}`)}
    >
      {checking && <LoaderCircle className="animate-spin" />}
      开始游戏
    </Button>
  );
}
