import { defineClientPlugin, defineRoutes } from "../../define-plugin";
import { route } from "../../route";
import type { Session } from "../../types";
import type { RequestMagicLinkInput, VerifyMagicLinkInput } from "./types";

export function magicLinkPlugin<TFields = unknown>() {
  const routes = defineRoutes(
    route<RequestMagicLinkInput, { message: string }>()({
      method: "POST",
      path: "/signin",
    }),
    route<VerifyMagicLinkInput, Session<TFields>>()({
      method: "GET",
      path: "/verify",
      parseSession: true,
    }),
  );

  return defineClientPlugin({
    id: "magic-link",
    basePath: "/magic-link",
    routes,
  });
}

export type { RequestMagicLinkInput, VerifyMagicLinkInput } from "./types";
