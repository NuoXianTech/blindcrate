/** 图片的用途, 对应后端 upload.PurposeAvatar 等; 头像登录用户都可以上传, 其他只有管理员可以上传 */
export type UploadPurpose = 'avatar' | 'item' | 'venue' | 'shop';

/** 上传结果, 对应后端 upload.View */
export interface UploadedImage {
  id: number;
  /** 保存到头像、物品等图片地址中的地址, 本站存储时为 /uploads/... */
  url: string;
  /** 保存后的格式, image/png 或 image/jpeg */
  content_type: string;
  /** 保存后的字节数 */
  size: number;
  width: number;
  height: number;
}
