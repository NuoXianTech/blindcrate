import type {PageParams} from '../core';

/** 配置包中的配置种类代码, 也是包中的文件名; 由后端各模块注册, 列表见 PackSectionInfo */
export type PackSection = string;

/** 一种配置, 对应后端 pack.SectionView; 列表按依赖排列, 导入时按这个顺序写入 */
export interface PackSectionInfo {
  code: PackSection;
  label: string;
  /** 导出时的说明 */
  hint: string;
  /** 是否可以改名导入 */
  renamable: boolean;
}

/** 一项配置与已有配置的比较结果: 新增、覆盖、相同、冲突 */
export type PackStatus = 'new' | 'changed' | 'same' | 'conflict';

/** 一项配置的处理方式: 新增或覆盖、跳过、以新代码导入 */
export type PackActionType = 'overwrite' | 'skip' | 'rename';

export interface ExportPackRequest {
  name: string;
  author: string;
  sections: PackSection[];
}

/** 配置包的说明, 对应后端 admin.PackManifest */
export interface PackManifest {
  format: string;
  /** 格式版本 */
  version: number;
  name: string;
  author: string;
  exported_at: string;
  sections: PackSection[];
}

/** 一项配置的处理结果, 对应后端 admin.PackEntryResult */
export interface PackEntryResult {
  section: PackSection;
  /** 包中的代码, 品质为等级, 物品为 ID, 设置为设置组的 key */
  key: string;
  name: string;
  /** 跳过的项为与已有配置比较的结果 */
  status: PackStatus;
  action: PackActionType;
  /** 改名导入时的新代码 */
  code?: string;
  /** 无法导入的原因 */
  reason?: string;
}

/** 一项配置的处理方式, 没有提交的项按覆盖处理 */
export interface PackAction {
  section: PackSection;
  key: string;
  action: PackActionType;
  code: string;
}

export interface PackPreview {
  manifest: PackManifest;
  /** 文件的 sha256 */
  digest: string;
  size: number;
  entries: PackEntryResult[];
}

/** 一次导入的记录, 对应后端 admin.Import */
export interface PackImport {
  id: number;
  user_id: number;
  name: string;
  author: string;
  version: number;
  digest: string;
  size: number;
  created: number;
  updated: number;
  /** 包括与已有配置相同的项 */
  skipped: number;
  /** 每一项的处理结果, 列表中没有 */
  entries?: PackEntryResult[];
  created_at: string;
}

export type ListPackImportsParams = PageParams;

export interface ListPackImportsData {
  total: number;
  imports: PackImport[];
}
