package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/techschool/simplebank/model"
	"github.com/techschool/simplebank/service"
)

type UserController struct {
	service service.UserService
}

func NewUserController(service *service.UserService) *UserController {
	return &UserController{
		service: *service,
	}
}

func (c *UserController) createUser(ctx *gin.Context) {
	var req model.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := c.service.CreateUser(ctx, req)
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

func (c *UserController) loginUser(ctx *gin.Context) {
	var req model.LoginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	req.UserAgent = ctx.Request.UserAgent()
	req.ClientIP = ctx.ClientIP()

	user, err := c.service.LoginUser(ctx, req)
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
