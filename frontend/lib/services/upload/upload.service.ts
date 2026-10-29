import {BaseService} from '../core';
import type {UploadPurpose, UploadedImage} from './types';

export class UploadService extends BaseService {
  protected static readonly basePath = '/uploads';

  /** 上传图片, 后端按用途缩小尺寸并重新编码, 返回的地址保存到对应的图片地址中 */
  static uploadImage(purpose: UploadPurpose, file: File) {
    const form = new FormData();
    form.append('purpose', purpose);
    form.append('file', file);
    // 指定为 multipart 避免按默认的 JSON 序列化, 浏览器会补上分隔符; 大图片上传较慢, 放宽超时
    return this.post<UploadedImage>('', form, {headers: {'Content-Type': 'multipart/form-data'}, timeout: 60000});
  }
}
