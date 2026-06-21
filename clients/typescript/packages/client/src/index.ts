export { VERSION } from "./version";

export { createAuthClient } from "./client";

export { deriveErrorCode, LimenError } from "./errors";
export type { LimenErrorCode } from "./errors";

export { camelizeEach, camelizeKeys } from "./helpers";
export { defaultSessionParse, normalizeUser } from "./normalize";

export type { SessionState, SessionStore } from "./session-store";

export type {
  AfterResponseHook,
  BeforeRequestHook,
  FetchInit,
  PluginClientOverride,
  PluginIdOf,
  PluginOverrides,
  RequestContext,
  ResponseContext,
  RouteMatcher,
} from "./plugin";

export { defineClientPlugin, defineRoutes } from "./define-plugin";
export { route } from "./route";
export { defaultSerialize } from "./serialize";

export type { AnyRouteContext, RouteContext } from "./context";
export type { RunRoute } from "./define-plugin";
export type { RouteCallOptions, RouteHandler } from "./route";

export { coreClientPlugin } from "./routes";
export type { ActiveSession, CoreContribution, VerifyEmailInput } from "./routes";

export type {
  AuthClient,
  ClientFetchOptions,
  CreateAuthClientOptions,
  EnvelopeConfig,
  EnvelopeFields,
  EnvelopeMode,
  HTTPMethod,
  ParseSession,
  RedirectFn,
  Session,
  User,
} from "./types";
