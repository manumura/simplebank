package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/techschool/simplebank/model"
	"github.com/techschool/simplebank/service"
)

type UserHandler struct {
	// https://stackoverflow.com/questions/28014591/nameless-fields-in-go-structs
	// service service.UserService
	service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{
		// service: *service,
		*service,
	}
}

func (h *UserHandler) createUser(ctx *gin.Context) {
	var req model.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// user, err := h.service.CreateUser(ctx, req)
	user, err := h.CreateUser(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, user)
}

func (h *UserHandler) loginUser(ctx *gin.Context) {
	var req model.LoginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	req.UserAgent = ctx.Request.UserAgent()
	req.ClientIP = ctx.ClientIP()

	// user, err := h.service.LoginUser(ctx, req)
	user, err := h.LoginUser(ctx, req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		if errors.Is(err, service.ErrInvalidPassword) {
			ctx.JSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, user)
}
