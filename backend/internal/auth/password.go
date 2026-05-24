package auth

import "golang.org/x/crypto/bcrypt"

// bcryptCost is the bcrypt work factor. 12 is the architecture-mandated value
// (~250ms hash on modern hardware — sufficiently slow against brute force).
const bcryptCost = 12

// HashPassword returns the bcrypt hash of plain.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword returns true when plain matches the given bcrypt hash.
// Constant-time comparison via bcrypt.CompareHashAndPassword.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
