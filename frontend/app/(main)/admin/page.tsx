import {redirect} from 'next/navigation';
import {ADMIN_NAV} from '@/lib/nav';

/** 管理区入口, 进入第一个管理页面 */
export default function AdminPage() {
  redirect(ADMIN_NAV[0].entries[0].href);
}
