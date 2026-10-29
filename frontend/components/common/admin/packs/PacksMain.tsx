'use client';

import {useState} from 'react';
import {Download, History, Upload} from 'lucide-react';
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs';
import {ExportPanel} from './ExportPanel';
import {ImportHistory} from './ImportHistory';
import {ImportPanel} from './ImportPanel';

/**
 * 导入导出: 导出配置包, 导入前预览差异并逐项选择处理方式, 以及导入记录
 */
export function PacksMain() {
  const [tab, setTab] = useState('export');
  // 导入后重新拉取导入记录
  const [imported, setImported] = useState(0);

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">导入导出</h1>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="export">
            <Download />
            导出
          </TabsTrigger>
          <TabsTrigger value="import">
            <Upload />
            导入
          </TabsTrigger>
          <TabsTrigger value="history">
            <History />
            导入记录
          </TabsTrigger>
        </TabsList>
        <TabsContent value="export" className="mt-2">
          <ExportPanel />
        </TabsContent>
        {/* 切换标签时保留已上传的配置包和选择的处理方式 */}
        <TabsContent value="import" className="mt-2" forceMount hidden={tab !== 'import'}>
          <ImportPanel
            onImported={() => {
              setImported((n) => n + 1);
              setTab('history');
            }}
          />
        </TabsContent>
        <TabsContent value="history" className="mt-2">
          <ImportHistory key={imported} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
