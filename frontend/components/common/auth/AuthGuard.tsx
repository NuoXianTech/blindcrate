"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
import { LoaderCircle } from "lucide-react";
import { useAuth } from "./AuthProvider";

/** 需要登录的页面: 未登录时跳转登录页, 登录后回到当前页面; 主动退出登录则回首页 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, loading, loggedOut } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (loading || user) return;
    router.replace(loggedOut ? "/" : `/login?redirect=${encodeURIComponent(pathname)}`);
  }, [loading, user, loggedOut, pathname, router]);

  if (!user) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <LoaderCircle className="size-6 animate-spin text-primary" />
      </div>
    );
  }
  return <>{children}</>;
}
