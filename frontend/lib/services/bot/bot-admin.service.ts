import {BaseService} from '../core';
import type {
  BotName,
  BotNameRequest,
  BotPersona,
  BotPersonaRequest,
  BotStrategy,
  CreateBotNamesData,
  CreateBotNamesRequest,
  ListBotNamesData,
  ListBotNamesParams,
  ListBotPersonasData,
  ListBotPersonasParams,
} from './types';

/** 人机的后台接口: 昵称库、风格和决策策略 */
export class BotAdminService extends BaseService {
  protected static readonly basePath = '/admin/bot';

  /** 昵称列表, 按 ID 倒序 */
  static listNames(params: ListBotNamesParams) {
    return this.get<ListBotNamesData>('/names', {params});
  }

  /** 批量新建昵称, 已存在的和本次重复的跳过 */
  static createNames(body: CreateBotNamesRequest) {
    return this.post<CreateBotNamesData>('/names', body);
  }

  static updateName(id: number, body: BotNameRequest) {
    return this.put<BotName>(`/names/${id}`, body);
  }

  /** 物理删除, 已开局的对局中保存的是昵称本身, 不受影响 */
  static deleteName(id: number) {
    return this.delete<null>(`/names/${id}`);
  }

  /** 风格列表, 按 ID 排列 */
  static listPersonas(params: ListBotPersonasParams) {
    return this.get<ListBotPersonasData>('/personas', {params});
  }

  static createPersona(body: BotPersonaRequest) {
    return this.post<BotPersona>('/personas', body);
  }

  /** 修改风格, 需提交全部字段, 代码不能修改; 进行中的对局不受影响 */
  static updatePersona(id: number, body: BotPersonaRequest) {
    return this.put<BotPersona>(`/personas/${id}`, body);
  }

  /** 软删除, 被地图引用时不能删除 */
  static deletePersona(id: number) {
    return this.delete<null>(`/personas/${id}`);
  }

  /** 决策策略, 按注册顺序 */
  static listStrategies() {
    return this.get<BotStrategy[]>('/strategies');
  }
}
