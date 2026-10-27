import { BaseService } from "../core";
import type {
  AdminUser,
  Currency,
  Item,
  ItemRequest,
  ListAuditLogsData,
  ListAuditLogsParams,
  ListItemsData,
  ListItemsParams,
  ListUsersData,
  ListUsersParams,
  SettingGroup,
  UpdateSettingRequest,
  UpdateUserRequest,
} from "./types";

export class AdminService extends BaseService {
  protected static readonly basePath = "/admin";

  static listUsers(params: ListUsersParams) {
    return this.get<ListUsersData>("/users", { params });
  }

  static updateUser(id: number, body: UpdateUserRequest) {
    return this.patch<Pick<AdminUser, "id" | "username" | "nickname" | "email" | "avatar_url" | "is_admin">>(
      `/users/${id}`,
      body,
    );
  }

  /** 封禁到指定时间(RFC3339), 并让其所有设备下线 */
  static banUser(id: number, bannedUntil: string, reason: string) {
    return this.put<null>(`/users/${id}/ban`, { banned_until: bannedUntil, reason });
  }

  static unbanUser(id: number) {
    return this.delete<null>(`/users/${id}/ban`);
  }

  /** 停用或启用, 停用时让其所有设备下线 */
  static setUserDisabled(id: number, disabled: boolean) {
    return this.put<null>(`/users/${id}/disabled`, { disabled });
  }

  static listAuditLogs(params: ListAuditLogsParams) {
    return this.get<ListAuditLogsData>("/audit-logs", { params });
  }

  static listCurrencies() {
    return this.get<{ currencies: Currency[] }>("/wallet/currencies");
  }

  static listItems(params: ListItemsParams) {
    return this.get<ListItemsData>("/items", { params });
  }

  static createItem(body: ItemRequest) {
    return this.post<Item>("/items", body);
  }

  static updateItem(id: number, body: ItemRequest) {
    return this.put<Item>(`/items/${id}`, body);
  }

  /** 删除后已持有的玩家背包保留为占位物品, 返回删除时的持有人数 */
  static deleteItem(id: number) {
    return this.delete<{ holder_count: number }>(`/items/${id}`);
  }

  static listSettings() {
    return this.get<{ groups: SettingGroup[] }>("/settings");
  }

  /** 保存一个设置组, 立即生效, 返回保存后的设置组 */
  static updateSetting(key: string, body: UpdateSettingRequest) {
    return this.put<SettingGroup>(`/settings/${key}`, body);
  }

  /** 用已保存的邮件设置发送测试邮件 */
  static testMail(to: string) {
    return this.post<null>("/settings/mail/test", { to });
  }
}
