import { defineClientPlugin, defineRoutes } from "../../define-plugin";
import { camelizeKeys } from "../../helpers";
import { route, type RouteHandler } from "../../route";
import type { LinkOAuthInput, OAuthAccount, OAuthAuthorizeResult, OAuthTokens, SignInOAuthInput } from "./types";

const fetchThenRedirect: RouteHandler<SignInOAuthInput, OAuthAuthorizeResult> = async (ctx, input, http) => {
  const { disableRedirect, ...rest } = input;
  const { url } = await http<{ url: string }>(rest);
  return { url, redirect: disableRedirect === true ? false : ctx.redirect(url) };
};

export function oauthClientPlugin() {
  const routes = defineRoutes(
    route<SignInOAuthInput, OAuthAuthorizeResult>()({
      method: "GET",
      path: "/:provider/authorize",
      as: "signIn.social",
      params: ["provider"],
      handler: fetchThenRedirect,
    }),
    route<LinkOAuthInput, OAuthAuthorizeResult>()({
      method: "GET",
      path: "/:provider/link",
      as: "social.link",
      params: ["provider"],
      handler: fetchThenRedirect,
    }),
    route<{ provider: string }, void>()({
      method: "DELETE",
      path: "/:provider/unlink",
      as: "social.unlink",
      params: ["provider"],
    }),
    route<void, OAuthAccount[]>()({
      method: "GET",
      path: "/accounts",
      as: "social.accounts",
    }),
    route<{ provider: string }, OAuthTokens>()({
      method: "GET",
      path: "/:provider/tokens",
      as: "social.tokens",
      params: ["provider"],
      parse: camelizeKeys,
    }),
    route<{ provider: string }, OAuthTokens>()({
      method: "POST",
      path: "/:provider/tokens/refresh",
      as: "social.refreshTokens",
      params: ["provider"],
      parse: camelizeKeys,
    }),
  );

  return defineClientPlugin({
    id: "oauth",
    basePath: "/oauth",
    routes,
  });
}

export type {
  LinkOAuthInput,
  OAuthAccount,
  OAuthAuthorizeQuery,
  OAuthAuthorizeResult,
  OAuthTokens,
  SignInOAuthInput,
} from "./types";
