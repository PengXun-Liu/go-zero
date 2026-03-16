package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/examples/farm/internal/svc"
	"github.com/zeromicro/go-zero/examples/farm/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// RegisterLogic handles user registration.
type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewRegisterLogic creates a new RegisterLogic.
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Register creates a new user account.
// For simplicity, credentials are stored in Redis. In production use a proper database.
func (l *RegisterLogic) Register(req *types.RegisterRequest) (*types.RegisterResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, errors.New("username and password are required")
	}

	userKey := fmt.Sprintf("user:name:%s", req.Username)

	// Check if the username already exists.
	exists, err := l.svcCtx.Redis.ExistsCtx(l.ctx, userKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if exists {
		return nil, errors.New("username already taken")
	}

	// Generate a simple user ID using timestamp.
	userId := fmt.Sprintf("user:%d", time.Now().UnixNano())

	// Persist the user record.
	if err := l.svcCtx.Redis.HmsetCtx(l.ctx, userKey, map[string]string{
		"userId":   userId,
		"password": req.Password,
	}); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	l.Infof("registered new user: %s (%s)", req.Username, userId)

	return &types.RegisterResponse{
		UserId:   userId,
		Username: req.Username,
	}, nil
}

// LoginLogic handles user login and JWT issuance.
type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewLoginLogic creates a new LoginLogic.
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Login validates user credentials and returns a signed JWT token.
func (l *LoginLogic) Login(req *types.LoginRequest) (*types.LoginResponse, error) {
	userKey := fmt.Sprintf("user:name:%s", req.Username)

	storedPassword, err := l.svcCtx.Redis.HgetCtx(l.ctx, userKey, "password")
	if err != nil || storedPassword == "" {
		return nil, errors.New("invalid username or password")
	}

	if storedPassword != req.Password {
		return nil, errors.New("invalid username or password")
	}

	userId, err := l.svcCtx.Redis.HgetCtx(l.ctx, userKey, "userId")
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	now := time.Now().Unix()
	expireAt := now + l.svcCtx.Config.Auth.AccessExpire

	token, err := generateToken(l.svcCtx.Config.Auth.AccessSecret, userId, expireAt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &types.LoginResponse{
		UserId:      userId,
		Username:    req.Username,
		AccessToken: token,
		ExpireAt:    expireAt,
	}, nil
}

// generateToken creates a signed JWT token with the given user ID.
func generateToken(secret, userId string, expireAt int64) (string, error) {
	claims := jwt.MapClaims{
		"userId": userId,
		"exp":    expireAt,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
