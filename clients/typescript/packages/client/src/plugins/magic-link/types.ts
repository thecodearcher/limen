export type RequestMagicLinkInput = {
  email: string;
  redirectUri?: string;
  newUserRedirectUri?: string;
  errorRedirectUri?: string;
  meta?: Record<string, unknown>;
};

export type VerifyMagicLinkInput = {
  token: string;
};
