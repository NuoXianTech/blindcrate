import { BaseService } from "../core";
import type { CredentialsRequest, UserInfo } from "./types";

export class AuthService extends BaseService {
  protected static readonly basePath = "/auth";

  static login(body: CredentialsRequest) {
    return this.post<UserInfo>("/login", body);
  }

  static register(body: CredentialsRequest) {
    return this.post<UserInfo>("/register", body);
  }

  static logout() {
    return this.post<null>("/logout");
  }

  static getUserInfo() {
    return this.get<UserInfo>("/user-info");
  }
}
