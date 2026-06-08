package service

import (
	"context"
	"errors"
	"time"

	"auction/internal/model"
	"auction/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken    = errors.New("username already registered")
	ErrInvalidLogin     = errors.New("invalid username or password")
	ErrUserDisabled     = errors.New("account is disabled")
	ErrAlreadyLoggedIn  = errors.New("ALREADY_LOGGED_IN")
)

type AuthSvc struct {
	userRepo  *repository.UserRepo
	jwtSecret string
	jwtExpire time.Duration
}

func NewAuthSvc(userRepo *repository.UserRepo, jwtSecret string, expireHours int) *AuthSvc {
	return &AuthSvc{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtExpire: time.Duration(expireHours) * time.Hour,
	}
}

type RegisterInput struct {
	Username string `json:"username" form:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" form:"password" binding:"required,min=6,max=32"`
	Nickname string `json:"nickname" form:"nickname" binding:"required,max=32"`
	Role     uint8  `json:"role" form:"role"`
	Avatar   string `json:"avatar"` // set manually after file upload, not bound from form
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Force    bool   `json:"force"` // force login, invalidate old sessions
}

type AuthResult struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

func (s *AuthSvc) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	existing, _ := s.userRepo.GetByUsername(ctx, input.Username)
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	if input.Role > 2 {
		input.Role = 0
	}

	user := &model.User{
		Username:     input.Username,
		PasswordHash: string(hash),
		Nickname:     input.Nickname,
		Avatar:       input.Avatar,
		Role:         model.UserRole(input.Role),
		Status:       model.UserStatusNormal,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Token: token, User: user}, nil
}

func (s *AuthSvc) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	user, err := s.userRepo.GetByUsername(ctx, input.Username)
	if err != nil {
		return nil, ErrInvalidLogin
	}

	if user.Status != model.UserStatusNormal {
		return nil, ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidLogin
	}

	// Check if already logged in elsewhere (last login was within 5 minutes)
	if user.LastLoginAt != nil && time.Since(*user.LastLoginAt) < 5*time.Minute && !input.Force {
		return nil, ErrAlreadyLoggedIn
	}

	// Bump token version & update last login
	user.TokenVersion++
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user.ID, map[string]any{
		"token_version": user.TokenVersion,
		"last_login_at": user.LastLoginAt,
	}); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Token: token, User: user}, nil
}

func (s *AuthSvc) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"nickname": user.Nickname,
		"avatar":   user.Avatar,
		"role":     uint8(user.Role),
		"ver":      user.TokenVersion,
		"exp":      time.Now().Add(s.jwtExpire).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
