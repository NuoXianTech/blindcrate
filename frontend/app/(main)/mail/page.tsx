import type {Metadata} from 'next';
import {MailMain} from '@/components/common/mail';

export const metadata: Metadata = {title: '邮件'};

export default function MailPage() {
  return <MailMain />;
}
