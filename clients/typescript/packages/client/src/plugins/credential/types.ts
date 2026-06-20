export type SignInCredentialInput = {
  /** Email or username, depending on what the server enables. */
  credential: string;
  password: string;
  /** When true (default), issues a long-lived session. */
  rememberMe?: boolean;
};

export type SignUpCredentialInput = {
  email: string;
  password: string;
  /** Required when username-on-signup is enabled. */
  username?: string;
  /** Optional profile fields the server records on the user. */
  firstname?: string;
  lastname?: string;
  /** Extra fields recorded on the user at registration. */
  additionalFields?: Record<string, unknown>;
};

export type ChangePasswordInput = {
  currentPassword: string;
  newPassword: string;
  /** Invalidate other sessions on success. Defaults to true. */
  revokeOtherSessions?: boolean;
};

export type SetPasswordInput = {
  newPassword: string;
  revokeOtherSessions?: boolean;
};

export type RequestPasswordResetInput = {
  email: string;
};

export type ResetPasswordInput = {
  token: string;
  newPassword: string;
};

export type CheckUsernameInput = {
  username: string;
};
