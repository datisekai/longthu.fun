package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"

	dbgen "github.com/datisekai/longthu.fun/backend/internal/db/generated"
)

// PublicUser is the auth response shape — never leaks `password_hash`.
type PublicUser struct {
	ID          uint64 `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Tier        string `json:"tier"`
}

func publicFrom(u dbgen.User) PublicUser {
	return PublicUser{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Tier:        u.Tier,
	}
}

// Service is the auth domain entry point. Hands sit on the supplied queries.
type Service struct {
	q *dbgen.Queries
}

// NewService builds a Service from a *sql.DB.
func NewService(db *sql.DB) *Service {
	return &Service{q: dbgen.New(db)}
}

// Register creates a new user. Returns ErrEmailExists if the email is taken.
func (s *Service) Register(ctx context.Context, email, password, displayName string) (PublicUser, error) {
	email = normalizeEmail(email)
	if email == "" {
		return PublicUser{}, fmt.Errorf("auth.Register: email required")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return PublicUser{}, fmt.Errorf("auth.Register: hash: %w", err)
	}

	res, err := s.q.InsertUser(ctx, dbgen.InsertUserParams{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
	})
	if err != nil {
		if isDuplicateEntry(err) {
			return PublicUser{}, ErrEmailExists
		}
		return PublicUser{}, fmt.Errorf("auth.Register: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return PublicUser{}, fmt.Errorf("auth.Register: lastInsertId: %w", err)
	}

	user, err := s.q.GetUserByID(ctx, uint64(id))
	if err != nil {
		return PublicUser{}, fmt.Errorf("auth.Register: fetch new user: %w", err)
	}
	return publicFrom(user), nil
}

// Login verifies email + password. Returns ErrInvalidCredentials for any
// failure (no enumeration).
func (s *Service) Login(ctx context.Context, email, password string) (PublicUser, error) {
	email = normalizeEmail(email)
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicUser{}, ErrInvalidCredentials
		}
		return PublicUser{}, fmt.Errorf("auth.Login: lookup: %w", err)
	}
	if !VerifyPassword(user.PasswordHash, password) {
		return PublicUser{}, ErrInvalidCredentials
	}
	return publicFrom(user), nil
}

// GetByID fetches the current user, used by the session middleware + /me.
func (s *Service) GetByID(ctx context.Context, id uint64) (PublicUser, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicUser{}, ErrUserNotFound
		}
		return PublicUser{}, fmt.Errorf("auth.GetByID: %w", err)
	}
	return publicFrom(user), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// isDuplicateEntry detects MySQL's 1062 error (unique constraint violation).
func isDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}
