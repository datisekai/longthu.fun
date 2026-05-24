package auth

import "errors"

// ErrEmailExists is returned by Register when the email is already registered.
var ErrEmailExists = errors.New("auth: email already registered")

// ErrInvalidCredentials is returned by Login for any of "no such email" or
// "wrong password". Single error type prevents user-existence enumeration
// (the handler maps both to the same 401 response).
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

// ErrUserNotFound is returned by the middleware when the JWT's `sub` no
// longer maps to a row (e.g. user hard-deleted while their cookie is alive).
var ErrUserNotFound = errors.New("auth: user no longer exists")
