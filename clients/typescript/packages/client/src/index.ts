export { createAuthClient } from "./client";

export { deriveErrorCode, LimenError } from "./errors";
export type { LimenErrorCode } from "./errors";

export { camelizeEach, camelizeKeys, camelizePage } from "./helpers";
export { defaultSessionParse, normalizeUser } from "./normalize";

export type { SessionState, SessionStore } from "./session-store";
export { createStore } from "./data-store";
export type { DataStore, RefetchOptions, StoreLoader, StoreState } from "./data-store";
export { createAuthStore } from "./auth-store";
export type { CreateAuthStoreOptions } from "./auth-store";

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

export { defineClientPlugin, defineFields, defineRoutes, defineStores, schema } from "./define-plugin";
export { route } from "./route";
export { defaultSerialize } from "./serialize";

export { can, canAny } from "./permissions";
export type { PermissionInput, PermissionSource } from "./permissions";

export { currentData, effect, storeRef } from "./stores";
export type { EffectTrigger, PluginStores, StoreEffect, StoreRef, StoreRegistry } from "./stores";
export { sessionStore } from "./session-store";

export type { AnyRouteContext, RouteContext } from "./context";
export type { DeclaredFields, FieldsOf, ModelFields, PluginSchema, RunRoute, StrictConfig } from "./define-plugin";
export type { RouteCallOptions, RouteHandler } from "./route";

export { coreClientPlugin } from "./routes";
export type { ActiveSession, CoreContribution, RequestEmailVerificationInput, VerifyEmailInput } from "./routes";

export type { CoreStores, StoresOf, StoreValues } from "./infer";

export type {
  AuthClient,
  ClientFetchOptions,
  CreateAuthClientOptions,
  EnvelopeConfig,
  EnvelopeFields,
  EnvelopeMode,
  HTTPMethod,
  Page,
  ParseSession,
  RedirectFn,
  Session,
  StoreAtoms,
  User,
} from "./types";
