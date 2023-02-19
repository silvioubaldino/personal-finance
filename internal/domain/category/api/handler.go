package api

import (
	"context"
	"net/http"
	"strconv"

	"personal-finance/internal/domain/category/service"
	"personal-finance/internal/model"
	"personal-finance/internal/plataform/authentication"

	"github.com/gin-gonic/gin"
)

type handler struct {
	srv         service.Service
	authService authentication.Auth
}

func NewCategoryHandlers(r *gin.Engine, srv service.Service, auth authentication.Auth) {
	handler := handler{
		srv:         srv,
		authService: auth,
	}

	r.GET("/ping", handler.ping())
	r.GET("/categories", handler.FindAll())
	r.GET("/categories/:id", handler.FindByID())
	r.POST("/categories", handler.Add())
	r.PUT("/categories/:id", handler.Update())
	r.DELETE("/categories/:id", handler.Delete())
}

func (h handler) ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	}
}

func (h handler) FindAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("user_token")
		userID, err := h.authService.ValidToken(userToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}
		categories, err := h.srv.FindAll(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, categories)
	}
}

func (h handler) FindByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("user_token")
		userID, err := h.authService.ValidToken(userToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}

		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		categ, err := h.srv.FindByID(c.Request.Context(), int(id), userID)
		if err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}
		c.JSON(http.StatusOK, categ)
	}
}

func (h handler) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("user_token")
		userID, err := h.authService.ValidToken(userToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}

		var categ model.Category
		err = c.ShouldBindJSON(&categ)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		savedCateg, err := h.srv.Add(context.Background(), categ, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusCreated, savedCateg)
	}
}

func (h handler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken := c.GetHeader("user_token")
		userID, err := h.authService.ValidToken(userToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			return
		}

		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		var category model.Category
		err = c.ShouldBindJSON(&category)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		updatedCateg, err := h.srv.Update(context.Background(), int(id), category, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, updatedCateg)
	}
}

func (h handler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		idString := c.Param("id")
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}
		err = h.srv.Delete(context.Background(), int(id))
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusNoContent, nil)
	}
}
