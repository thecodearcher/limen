import { defineClientPlugin, defineRoutes } from "../../define-plugin";
import { route } from "../../route";
import type { Session } from "../../types";
import type {
  DisableTwoFactorInput,
  FinalizeSetupInput,
  InitiateSetupInput,
  SendOTPInput,
  TwoFactorConfig,
  TwoFactorSetupURI,
  VerifyInput,
} from "./types";

export function twoFactorPlugin<TFields = unknown>(config: TwoFactorConfig) {
  const routes = defineRoutes(
    route<InitiateSetupInput, TwoFactorSetupURI>()({
      method: "POST",
      path: "/initiate-setup",
    }),
    route<FinalizeSetupInput, Session<TFields>>()({
      method: "POST",
      path: "/finalize-setup",
      parseSession: true,
    }),
    route<DisableTwoFactorInput, Session<TFields>>()({
      method: "POST",
      path: "/disable",
      parseSession: true,
    }),
    route<VerifyInput, Session<TFields>>()({
      method: "POST",
      path: "/verify",
      parseSession: true,
    }),
    route<void, TwoFactorSetupURI>()({
      method: "GET",
      path: "/totp/uri",
      as: "twoFactor.getTotpUri",
    }),
    route<void, string[]>()({
      method: "GET",
      path: "/backup-codes",
      as: "twoFactor.getBackupCodes",
    }),
    route<void, string[]>()({
      method: "PUT",
      path: "/backup-codes",
      as: "twoFactor.regenerateBackupCodes",
    }),
    route<SendOTPInput, { message: string }>()({
      method: "POST",
      path: "/otp/send",
      as: "twoFactor.sendOTP",
    }),
  );

  return defineClientPlugin({
    id: "two-factor",
    basePath: "/two-factor",
    routes,
    hooks: {
      afterResponse: [
        {
          match: "/signin/credential",
          run: (ctx) => {
            const body = ctx.body as Record<string, unknown>;
            if (body["two_factor_required"] === true) {
              config.onTwoFactorRedirect();
            }
            return ctx;
          },
        },
      ],
    },
  });
}

export type {
  DisableTwoFactorInput,
  FinalizeSetupInput,
  InitiateSetupInput,
  SendOTPInput,
  TwoFactorConfig,
  TwoFactorMethod,
  TwoFactorSetupURI,
  VerifyInput,
} from "./types";
