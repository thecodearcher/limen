import { defineRoutes } from "../../define-plugin";
import { route } from "../../route";
import type { Page, Session } from "../../types";
import { serializeBeginQuery, serializeFinishRegistration } from "./helpers";
import type {
  BeginAuthenticationInput,
  BeginAuthenticationResult,
  BeginRegistrationInput,
  BeginRegistrationResult,
  DeletePasskeyInput,
  FinishAuthenticationInput,
  FinishRegistrationInput,
  FinishRegistrationResult,
  ListPasskeysInput,
  Passkey,
  UpdatePasskeyInput,
} from "./types";

export function buildPasskeyRoutes<TFields>() {
  const beginRegistration = route<BeginRegistrationInput, BeginRegistrationResult>()({
    method: "GET",
    path: "/begin-registration",
    as: "passkey.beginRegistration",
    serialize: serializeBeginQuery,
  });

  const finishRegistration = route<FinishRegistrationInput, FinishRegistrationResult<TFields>>()({
    method: "POST",
    path: "/finish-registration",
    as: "passkey.finishRegistration",
    serialize: serializeFinishRegistration,
    parseSession: true,
  });

  const beginAuthentication = route<BeginAuthenticationInput, BeginAuthenticationResult>()({
    method: "GET",
    path: "/begin-authentication",
    as: "passkey.beginAuthentication",
    serialize: serializeBeginQuery,
  });

  const finishAuthentication = route<FinishAuthenticationInput, Session<TFields>>()({
    method: "POST",
    path: "/finish-authentication",
    as: "passkey.finishAuthentication",
    serialize: (input) => input,
    parseSession: true,
  });

  const deletePasskey = route<Pick<DeletePasskeyInput, "id">, void>()({
    method: "DELETE",
    path: "/:id",
    expose: false,
    params: ["id"],
  });

  return {
    beginRegistration,
    finishRegistration,
    beginAuthentication,
    finishAuthentication,
    deletePasskey,
    routes: defineRoutes(
      beginRegistration,
      finishRegistration,
      beginAuthentication,
      finishAuthentication,
      route<ListPasskeysInput, Page<Passkey>>()({
        method: "GET",
        path: "/",
        as: "passkey.list",
      }),
      route<UpdatePasskeyInput, Passkey>()({
        method: "PATCH",
        path: "/:id",
        as: "passkey.update",
        params: ["id"],
      }),
      deletePasskey,
    ),
  };
}

export type PasskeyRouteSet<TFields = unknown> = ReturnType<typeof buildPasskeyRoutes<TFields>>;
