package emailpassword

type ConfigOption func(*config)

// WithPasswordMinLength sets the minimum length of the password
func WithPasswordMinLength(passwordMinLength int) ConfigOption {
	return func(c *config) {
		c.passwordMinLength = passwordMinLength
	}
}

// WithPasswordRequireUppercase sets whether to require uppercase letters in the password
func WithPasswordRequireUppercase(passwordRequireUppercase bool) ConfigOption {
	return func(c *config) {
		c.passwordRequireUppercase = passwordRequireUppercase
	}
}

// WithPasswordRequireNumbers sets whether to require numbers in the password
func WithPasswordRequireNumbers(passwordRequireNumbers bool) ConfigOption {
	return func(c *config) {
		c.passwordRequireNumbers = passwordRequireNumbers
	}
}

// WithPasswordRequireSymbols sets whether to require symbols in the password
func WithPasswordRequireSymbols(passwordRequireSymbols bool) ConfigOption {
	return func(c *config) {
		c.passwordRequireSymbols = passwordRequireSymbols
	}
}

// WithHashFn sets the function to hash the password
func WithHashFn(hashFn func(password string) (string, error)) ConfigOption {
	return func(c *config) {
		c.hashFn = hashFn
	}
}

// WithCompareFn sets the function to compare the password and the hash
func WithCompareFn(compareFn func(password string, hash string) (bool, error)) ConfigOption {
	return func(c *config) {
		c.compareFn = compareFn
	}
}

// WithPasswordHasherConfigOptions sets the Argon2id configuration for the password hasher
func WithPasswordHasherConfigOptions(opts ...PasswordHasherConfigOption) ConfigOption {
	return func(c *config) {
		c.passwordHasherConfig = DefaultPasswordHasherConfig(opts...)
	}
}
