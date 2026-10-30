import type {Metadata} from 'next';
import {MailsMain} from '@/components/common/admin/mails';

export const metadata: Metadata = {title: '游戏邮件'};

export default function AdminMailsPage() {
  return <MailsMain />;
}
