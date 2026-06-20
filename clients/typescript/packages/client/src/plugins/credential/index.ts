import { defineClientPlugin, defineRoutes } from "../../define-plugin";
import { route } from "../../route";
import type { Session } from "../../types";
import type {
  ChangePasswordInput,
  CheckUsernameInput,
  RequestPasswordResetInput,
  ResetPasswordInput,
  SetPasswordInput,
  SignInCredentialInput,
  SignUpCredentialInput,
} from "./types";

export function credentialPasswordPlugin<TFields = unknown>() {
  const routes = defineRoutes(
    route<SignInCredentialInput, Session<TFields>>()({
      method: "POST",
      path: "/signin/credential",
      parseSession: true,
      as: "signIn.credential",
      defaults: { rememberMe: true },
    }),
    route<SignUpCredentialInput, Session<TFields>>()({
      method: "POST",
      path: "/signup/credential",
      parseSession: true,
      as: "signUp.credential",
    }),
    route<RequestPasswordResetInput, { message: string }>()({
      method: "POST",
      path: "/passwords/request-reset",
      as: "password.requestReset",
    }),
    route<ResetPasswordInput, { message: string }>()({
      method: "POST",
      path: "/passwords/reset",
      as: "password.reset",
    }),
    route<ChangePasswordInput, Session<TFields>>()({
      method: "POST",
      path: "/passwords/change",
      parseSession: true,
      defaults: { revokeOtherSessions: true },
      as: "password.change",
    }),
    route<SetPasswordInput, Session<TFields>>()({
      method: "PUT",
      path: "/passwords",
      as: "password.set",
      parseSession: true,
      defaults: { revokeOtherSessions: true },
    }),
    route<CheckUsernameInput, boolean>()({
      method: "POST",
      path: "/usernames/check",
      parse: (raw) => (raw as { available: boolean }).available,
      as: "username.checkAvailability",
    }),
  );

  return defineClientPlugin({
    id: "credential-password",
    basePath: "/",
    routes,
  });
}

export type {
  ChangePasswordInput,
  CheckUsernameInput,
  RequestPasswordResetInput,
  ResetPasswordInput,
  SetPasswordInput,
  SignInCredentialInput,
  SignUpCredentialInput,
} from "./types";
