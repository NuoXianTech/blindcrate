import { BaseService } from "../core";
import type {
  AdminUser,
  ListAuditLogsData,
  ListAuditLogsParams,
  ListUsersData,
  ListUsersParams,
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
}
