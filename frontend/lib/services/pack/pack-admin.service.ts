import {BaseService, apiClient} from '../core';
import type {
  ExportPackRequest,
  ListPackImportsData,
  ListPackImportsParams,
  PackAction,
  PackImport,
  PackPreview,
  PackSectionInfo,
} from './types';

/** 上传配置包的请求配置: 指定为 multipart 避免按默认的 JSON 序列化, 浏览器会补上分隔符; 配置包较大时较慢, 放宽超时 */
const PACK_UPLOAD = {headers: {'Content-Type': 'multipart/form-data'}, timeout: 120000};

/** 配置包和处理方式的上传表单 */
function packForm(file: File, actions: PackAction[]) {
  const form = new FormData();
  form.append('file', file);
  form.append('actions', JSON.stringify(actions));
  return form;
}

export class PackAdminService extends BaseService {
  protected static readonly basePath = '/admin';

  /** 全部配置种类, 按依赖排列 */
  static listPackSections() {
    return this.get<PackSectionInfo[]>('/packs/sections');
  }

  /** 导出配置包, 返回 zip 文件和下载时的文件名 */
  static async exportPack(body: ExportPackRequest) {
    // 图片较多时打包较慢, 放宽超时
    const res = await apiClient.post<Blob>(this.url('/packs/export'), body, {responseType: 'blob', timeout: 120000});
    const match = /filename="([^"]+)"/.exec(String(res.headers['content-disposition'] ?? ''));
    return {blob: res.data, filename: match?.[1] ?? 'blindcrate-pack.zip'};
  }

  /** 按处理方式试着导入, 返回每一项的比较结果, 不写入任何内容 */
  static previewPack(file: File, actions: PackAction[]) {
    return this.post<PackPreview>('/packs/preview', packForm(file, actions), PACK_UPLOAD);
  }

  /** 在一个事务中全部导入; 有未跳过的冲突项时返回 409, 错误的 data 为每一项的结果 */
  static importPack(file: File, actions: PackAction[]) {
    return this.post<PackImport>('/packs/import', packForm(file, actions), PACK_UPLOAD);
  }

  /** 从新到旧, 不含每一项的结果 */
  static listPackImports(params: ListPackImportsParams) {
    return this.get<ListPackImportsData>('/packs/imports', {params});
  }

  static getPackImport(id: number) {
    return this.get<PackImport>(`/packs/imports/${id}`);
  }
}
