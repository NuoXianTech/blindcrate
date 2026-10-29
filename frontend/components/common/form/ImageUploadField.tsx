'use client';

import {useRef, useState} from 'react';
import {LoaderCircle, Upload} from 'lucide-react';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Input} from '@/components/ui/input';
import {Label} from '@/components/ui/label';
import {useAuth} from '@/components/common/auth';
import {useSite} from '@/components/common/site';
import services from '@/lib/services';
import {toErrorMessage} from '@/lib/services/core';
import type {UploadPurpose} from '@/lib/services/upload';
import type {PublicSettings} from '@/lib/site';

/**
 * 图片上传字段的Props接口
 */
interface ImageUploadFieldProps {
  id: string;
  label: string;
  /** 图片的用途, 决定上传权限和保存的尺寸 */
  purpose: UploadPurpose;
  value: string;
  onChange: (url: string) => void;
  /** 输入框左侧的预览, 由调用方按展示方式渲染 */
  preview?: React.ReactNode;
  /** 输入框下方的说明 */
  help?: string;
  placeholder?: string;
  disabled?: boolean;
}

/** 可以上传的格式和提示中的名称, 与后端 upload.checkFormat 一致 */
const FORMATS = [
  {type: 'image/png', name: 'PNG', enabled: (upload: PublicSettings['upload']) => upload.allow_png},
  {type: 'image/jpeg', name: 'JPEG', enabled: (upload: PublicSettings['upload']) => upload.allow_jpeg},
  {type: 'image/webp', name: 'WebP', enabled: (upload: PublicSettings['upload']) => upload.allow_webp},
];

/**
 * 图片地址字段: 可以直接填写 http(s) 链接, 也可以上传图片后自动填入地址
 * 设置中关闭了头像上传或没有允许任何格式时只能填写链接
 */
export function ImageUploadField({id, label, purpose, value, onChange, preview, help, placeholder, disabled}: ImageUploadFieldProps) {
  const {upload} = useSite();
  const {user} = useAuth();
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  // 设置中允许的格式, 用于选择文件时过滤和上传前检查
  const formats = FORMATS.filter((f) => f.enabled(upload));
  const uploadable = formats.length > 0 && (purpose !== 'avatar' || upload.avatar_enabled || !!user?.is_admin);

  const handleFile = async (file: File | undefined) => {
    // 清空以便再次选择同一个文件
    if (inputRef.current) inputRef.current.value = '';
    if (!file) return;
    // 后端按内容识别格式, 这里只按浏览器给出的类型和大小提前拦下明显不符合的文件
    if (!formats.some((f) => f.type === file.type)) {
      toast.error(`只支持 ${formats.map((f) => f.name).join('、')} 格式的图片`);
      return;
    }
    if (file.size > upload.max_size_kb * 1024) {
      toast.error(`图片不能超过 ${upload.max_size_kb} KB`);
      return;
    }
    setUploading(true);
    try {
      const image = await services.upload.uploadImage(purpose, file);
      onChange(image.url);
      toast.success(`图片已上传 (${image.width}×${image.height}), 保存后生效`);
    } catch (err) {
      toast.error(toErrorMessage(err));
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex items-center gap-3">
        {preview}
        <Input
          id={id}
          placeholder={placeholder}
          maxLength={1024}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled || uploading}
        />
        {uploadable && (
          <>
            <input
              ref={inputRef}
              type="file"
              accept={formats.map((f) => f.type).join(',')}
              className="hidden"
              onChange={(e) => void handleFile(e.target.files?.[0])}
            />
            <Button
              type="button"
              variant="outline"
              className="shrink-0"
              onClick={() => inputRef.current?.click()}
              disabled={disabled || uploading}
            >
              {uploading ? <LoaderCircle className="animate-spin" /> : <Upload />}
              上传
            </Button>
          </>
        )}
      </div>
      {help && <span className="text-xs text-muted-foreground">{help}</span>}
    </div>
  );
}
