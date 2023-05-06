package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
	db "github.com/techschool/simplebank/db/sqlc"
	"github.com/techschool/simplebank/model"
	"github.com/techschool/simplebank/token"
	"github.com/techschool/simplebank/util"
)

type UserService interface {
	CreateUser(ctx context.Context, req model.CreateUserRequest) (model.UserResponse, error)
	LoginUser(ctx context.Context, req model.LoginUserRequest) (model.LoginUserResponse, error)
}

// TODO https://github.com/kyleconroy/sqlc/issues/710
// https://github.com/kyleconroy/sqlc/discussions/711
type UserServiceImpl struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}

func NewUserService(config util.Config, store db.Store, tokenMaker token.Maker) UserService {
	return &UserServiceImpl{
		store:      store,
		config:     config,
		tokenMaker: tokenMaker,
	}
}

// TODO custom error handling : https://golangbot.com/custom-errors/
// https://gobyexample.com/errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidPassword   = errors.New("username or password is invalid")
)

func (service *UserServiceImpl) CreateUser(ctx context.Context, req model.CreateUserRequest) (model.UserResponse, error) {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return model.UserResponse{}, err
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := service.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return model.UserResponse{}, ErrUserAlreadyExists
			}
		}
		return model.UserResponse{}, err
	}

	return model.UserResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}, nil
}

func (service *UserServiceImpl) LoginUser(ctx context.Context, req model.LoginUserRequest) (model.LoginUserResponse, error) {
	user, err := service.store.GetUser(ctx, req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.LoginUserResponse{}, ErrUserNotFound
		}
		return model.LoginUserResponse{}, err
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		return model.LoginUserResponse{}, ErrInvalidPassword
	}

	accessToken, accessPayload, err := service.tokenMaker.CreateToken(
		user.Username,
		service.config.AccessTokenDuration,
	)
	if err != nil {
		return model.LoginUserResponse{}, err
	}

	refreshToken, refreshPayload, err := service.tokenMaker.CreateToken(
		user.Username,
		service.config.RefreshTokenDuration,
	)
	if err != nil {
		return model.LoginUserResponse{}, err
	}

	session, err := service.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    req.UserAgent,
		ClientIp:     req.ClientIP,
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		return model.LoginUserResponse{}, err
	}

	return model.LoginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  model.ToUserResponse(user),
	}, nil
}
