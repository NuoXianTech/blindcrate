import {AdminService} from './admin';
import {AuthService} from './auth';
import {InventoryService} from './inventory';
import {WalletService} from './wallet';

/**
 * 服务层统一入口, 组件通过 services.xxx 调用接口
 *
 * 目录结构:
 *   /core - 基础工具
 *     - api-client.ts - Axios 实例和统一错误处理
 *     - base.service.ts - 服务基类, 自动剥掉响应外壳
 *     - types.ts - 通用类型
 *     - ws-client.ts - WebSocket 长连接, 订阅推送和发送请求
 *   /<模块> - 每个后端模块一个目录
 *     - xxx.service.ts - 服务实现, 继承 BaseService 并设置 basePath
 *     - types.ts - 请求和响应类型
 *     - index.ts - 导出服务和类型
 *
 * 使用示例:
 *   import services from '@/lib/services';
 *   const {balances} = await services.wallet.listBalances();
 *
 * 新增模块: 按上面的结构建目录, 再在下面的 services 对象里注册
 */
const services = {
  /** 登录、注册、个人资料 */
  auth: AuthService,
  /** 钱包余额和流水 */
  wallet: WalletService,
  /** 背包物品和记录 */
  inventory: InventoryService,
  /** 管理后台 */
  admin: AdminService,
};

export default services;
