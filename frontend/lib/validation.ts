/**
 * 表单校验规则, 与后端 internal/apps/auth/utils.go 的校验和 err.go 的文案保持一致
 * 返回错误文案, 通过返回 null
 */

const USERNAME_PATTERN = /^[A-Za-z0-9_]{3,32}$/;

export function checkUsername(username: string): string | null {
  return USERNAME_PATTERN.test(username) ? null : "用户名需为 3-32 位字母、数字或下划线";
}

/** bcrypt 最多处理 72 字节, 按字节计 */
export function checkPassword(password: string): string | null {
  const bytes = new TextEncoder().encode(password).length;
  return bytes >= 8 && bytes <= 72 ? null : "密码长度需为 8-72 位";
}

/** 按字符计, 调用方先去掉首尾空白 */
export function checkNickname(nickname: string): string | null {
  const length = [...nickname].length;
  return length >= 1 && length <= 32 ? null : "昵称需为 1-32 个字符";
}

export function checkPasswordConfirm(password: string, confirm: string): string | null {
  return password === confirm ? null : "两次输入的密码不一致";
}
