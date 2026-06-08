package errcode

const (
	Success = 0

	// Client errors 1xxxx
	ErrInvalidParam   = 40001
	ErrUnauthorized   = 40101
	ErrTokenExpired   = 40102
	ErrForbidden      = 40301
	ErrNotFound       = 40401
	ErrConflict       = 40901
	ErrTooManyRequest = 42901

	// Server errors 5xxxx
	ErrInternal = 50001
	ErrDatabase = 50002
	ErrThirdParty = 50003
)

var messages = map[int]string{
	Success:          "ok",
	ErrInvalidParam:  "invalid parameter",
	ErrUnauthorized:  "unauthorized",
	ErrTokenExpired:  "token expired",
	ErrForbidden:     "forbidden",
	ErrNotFound:      "not found",
	ErrConflict:      "resource conflict",
	ErrTooManyRequest: "too many requests",
	ErrInternal:      "internal server error",
	ErrDatabase:      "database error",
	ErrThirdParty:    "third party service error",
}

func Message(code int) string {
	if msg, ok := messages[code]; ok {
		return msg
	}
	return "unknown error"
}
