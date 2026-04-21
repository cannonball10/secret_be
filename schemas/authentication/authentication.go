package authentication

type AuthenticationProvider string

const (
	AuthenticationProvider_Clerk AuthenticationProvider = "CLERK"
	// Guest is a device-scoped identity. The "authentication ID" is
	// the client's random deviceId from localStorage — opaque, unverified,
	// and usable only as long as that device retains its storage. Guests
	// still get real User rows (and therefore real Passports), so stats
	// accrue locally even before sign-in.
	AuthenticationProvider_Guest AuthenticationProvider = "GUEST"
)
