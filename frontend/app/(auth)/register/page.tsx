import type {Metadata} from 'next';
import {Suspense} from 'react';
import {AuthForm} from '@/components/common/auth';

export const metadata: Metadata = {title: '注册'};

export default function RegisterPage() {
  // AuthForm 读取 searchParams, 需要包在 Suspense 里
  return (
    <Suspense>
      <AuthForm mode="register" />
    </Suspense>
  );
}
