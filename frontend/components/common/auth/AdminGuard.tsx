"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "./AuthProvider";

/** 管理员页面: 放在 AuthGuard 里面, 非管理员回到游戏首页 */
export function AdminGuard({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (user && !user.is_admin) router.replace("/home");
  }, [user, router]);

  if (!user?.is_admin) return null;
  return <>{children}</>;
}
