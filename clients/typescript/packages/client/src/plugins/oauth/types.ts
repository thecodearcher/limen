export type OAuthAuthorizeQuery = {
  /** Where the server redirects the browser after a successful callback. */
  redirectUri?: string;
  /** Where the server redirects on a failed callback. Falls back to `redirectUri`. */
  errorRedirectUri?: string;
};

export type SignInOAuthInput = OAuthAuthorizeQuery & {
  /** Provider id, e.g. `"google"` or `"github"`. */
  provider: string;
  /**
   * When true, skip auto-navigation and only resolve with the authorization
   * URL — the caller is responsible for navigating the browser.
   */
  disableRedirect?: boolean;
};

export type LinkOAuthInput = SignInOAuthInput;

export type OAuthAuthorizeResult = {
  /** Provider authorization URL. */
  url: string;
  /** Whether the SDK navigated to `url`. */
  redirect: boolean;
};

export type OAuthAccount = {
  provider: string;
  providerAccountId: string;
  scopes: string[];
  accessTokenExpiresAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type OAuthTokens = {
  accessToken: string;
  refreshToken?: string;
  idToken?: string;
  accessTokenExpiresAt?: string;
  scope?: string;
};
