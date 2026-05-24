// Package httpx provides shared HTTP plumbing — Problem Details responses,
// request helpers, etc.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Problem is the RFC 7807 problem-details payload shape this app emits.
// All non-2xx responses use it (per architecture §API & Communication).
type Problem struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

// FieldError is the per-field validation error shape inside Problem.Errors.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

const typeBase = "https://longthu.fun/errors/"

// Reply writes an RFC 7807 Problem Details JSON response with the given
// status, title, and optional detail. Instance is auto-set from the request path.
// Pass "" for type to default to a stable URL per status (e.g. .../validation).
func Reply(c *gin.Context, status int, title, detail string) {
	c.JSON(status, Problem{
		Type:     defaultType(status),
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: c.Request.URL.Path,
	})
}

// ReplyValidation writes a 422 with per-field errors.
func ReplyValidation(c *gin.Context, title string, errs []FieldError) {
	c.JSON(http.StatusUnprocessableEntity, Problem{
		Type:     typeBase + "validation",
		Title:    title,
		Status:   http.StatusUnprocessableEntity,
		Instance: c.Request.URL.Path,
		Errors:   errs,
	})
}

func defaultType(status int) string {
	switch status {
	case http.StatusBadRequest:
		return typeBase + "bad-request"
	case http.StatusUnauthorized:
		return typeBase + "unauthorized"
	case http.StatusForbidden:
		return typeBase + "forbidden"
	case http.StatusNotFound:
		return typeBase + "not-found"
	case http.StatusConflict:
		return typeBase + "conflict"
	case http.StatusUnprocessableEntity:
		return typeBase + "validation"
	case http.StatusTooManyRequests:
		return typeBase + "rate-limited"
	default:
		return typeBase + "internal"
	}
}
