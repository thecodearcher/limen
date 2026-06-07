package passkey

import "github.com/go-webauthn/webauthn/protocol"

// authSelectionOrDefault returns the configured AuthenticatorSelection
// criteria or a sensible default that mirrors better-auth: prefer a
// resident (passkey-capable) key and prefer user verification, but
// don't make either mandatory unless the caller opts in.
func authSelectionOrDefault(sel *protocol.AuthenticatorSelection) protocol.AuthenticatorSelection {
	if sel != nil {
		return *sel
	}
	return protocol.AuthenticatorSelection{
		ResidentKey:      protocol.ResidentKeyRequirementPreferred,
		UserVerification: protocol.VerificationPreferred,
	}
}
