import {dirname} from 'path';
import {fileURLToPath} from 'url';
import {FlatCompat} from '@eslint/eslintrc';
import nextVitals from 'eslint-config-next/core-web-vitals';
import nextTs from 'eslint-config-next/typescript';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const compat = new FlatCompat({
  baseDirectory: __dirname,
});

const eslintConfig = [
  // shadcn 等 CLI 生成的组件保持原样, 方便以后重新生成和对比上游
  {ignores: ['components/ui/**']},
  // eslint-config-next 16 只提供 flat config, 不能再用 compat.extends 引入
  ...nextVitals,
  ...nextTs,
  ...compat.extends('google'),
  {
    rules: {
      'valid-jsdoc': 'off',
      'require-jsdoc': 'off',
      'max-len': 'off',
      // 'linebreak-style': 'off', // 如果您是windows系统，可在开发时禁用行尾换行符检查
    },
  },
];

export default eslintConfig;
