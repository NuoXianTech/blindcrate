import type { Metadata } from "next";
import { Suspense } from "react";
import { AuthForm } from "@/components/common/auth/AuthForm";

export const metadata: Metadata = { title: "登录" };

export default function LoginPage() {
  // AuthForm 读取 searchParams, 需要包在 Suspense 里
  return (
    <Suspense>
      <AuthForm mode="login" />
    </Suspense>
  );
}
