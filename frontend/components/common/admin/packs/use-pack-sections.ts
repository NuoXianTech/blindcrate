'use client';

import {useEffect, useState} from 'react';
import services from '@/lib/services';
import type {PackSectionInfo} from '@/lib/services/admin';

/** 配置种类在页面会话中只拉取一次, 失败时下次使用再重试 */
let cache: Promise<PackSectionInfo[]> | null = null;

function load() {
  if (!cache) {
    cache = services.admin.listPackSections();
    cache.catch(() => {
      cache = null;
    });
  }
  return cache;
}

/** 后端注册的配置种类, 按依赖排列; 加载完成前为空数组 */
export function usePackSections() {
  const [sections, setSections] = useState<PackSectionInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    load()
        .then((list) => {
          if (alive) setSections(list);
        })
        .catch(() => {
          // 名称只用于展示, 失败时显示代码; 导出面板中没有可选的种类
        })
        .finally(() => {
          if (alive) setLoading(false);
        });
    return () => {
      alive = false;
    };
  }, []);

  return {sections, loading};
}

/** 配置种类的名称, 没有时显示代码 */
export function sectionLabel(sections: PackSectionInfo[], code: string) {
  return sections.find((s) => s.code === code)?.label ?? code;
}
