export type TwoFactorConfig = {
  /** Called when sign-in requires a two-factor challenge. */
  onTwoFactorRedirect: () => void;
};

export type VerifyInput = {
  code: string;
  method?: TwoFactorMethod;
};

export type InitiateSetupInput = {
  password: string;
};

export type FinalizeSetupInput = {
  code: string;
};

export type DisableTwoFactorInput = {
  password: string;
};

export type SendOTPInput = {
  email: string;
};

export type TwoFactorSetupURI = {
  uri: string;
};

export type TwoFactorMethod = "totp" | "otp";
