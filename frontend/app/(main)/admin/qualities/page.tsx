import type {Metadata} from 'next';
import {QualitiesMain} from '@/components/common/admin/qualities';

export const metadata: Metadata = {title: '品质管理'};

export default function AdminQualitiesPage() {
  return <QualitiesMain />;
}
