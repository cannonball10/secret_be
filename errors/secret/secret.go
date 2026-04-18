package secret

import "errors"

var (
	ErrSecretKeyEmpty            = errors.New("secret key is empty")
	ErrSecretClientUninitialized = errors.New("secretsmanager client is not initialized")
)
