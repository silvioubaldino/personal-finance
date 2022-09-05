package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"personal-finance/internal/business/service/car"
)

type handler struct {
	srv car.Service
}

func AddHandlers(r *gin.Engine, svc car.Service) {
	handler := handler{svc}

	r.GET("/ping", handler.ping())
	/*	r.GET("/cars", handler.FindAll())
		r.GET("/car/{id}", handler.FindByID())
		r.POST("/cars", handler.Add())
		r.PUT("/cars/{id}", handler.Update())
		r.DELETE("/cars/{id}", handler.Delete())*/
}

func (h handler) ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	}
}

/*func (h handler) Add(w http.ResponseWriter, r *http.Request) error {
	c := model.Car{}
	if err := web.DecodeJSON(r, &c); err != nil {
		return handlerError(w, err)
	}

	result, err := h.Service.Add(r.Context(), c)
	if err != nil {
		return handlerError(w, err)
	}
	return web.EncodeJSON(w, result, http.StatusCreated)
}

func (h handler) FindAll(w http.ResponseWriter, r *http.Request) error {
	cars, err := h.Service.FindAll(r.Context())
	if err != nil {
		return handlerError(w, err)
	}
	return web.EncodeJSON(w, cars, http.StatusOK)
}

func (h handler) FindByID(w http.ResponseWriter, r *http.Request) error {
	id, err := web.Params(r).String("id")
	if err != nil {
		return handlerError(w, err)
	}
	cars, err := h.Service.FindByID(r.Context(), id)
	if err != nil {
		return handlerError(w, err)
	}
	return web.EncodeJSON(w, cars, http.StatusOK)
}

func (h handler) Update(w http.ResponseWriter, r *http.Request) error {
	id, err := web.Params(r).String("id")
	if err != nil {
		return handlerError(w, err)
	}
	c := model.Car{}
	if err = web.DecodeJSON(r, &c); err != nil {
		return handlerError(w, err)
	}
	cars, err := h.Service.Update(r.Context(), id, c)
	if err != nil {
		return handlerError(w, err)
	}
	return web.EncodeJSON(w, cars, http.StatusOK)
}

func (h handler) Delete(w http.ResponseWriter, r *http.Request) error {
	id, err := web.Params(r).String("id")
	if err != nil {
		return handlerError(w, err)
	}
	err = h.Service.Delete(r.Context(), id)
	if err != nil {
		return handlerError(w, err)
	}
	return web.EncodeJSON(w, "success", http.StatusOK)
}

func handlerError(w http.ResponseWriter, err error) error {
	var customError model.BusinessError
	if errors.As(err, &customError) {
		return web.EncodeJSON(w, customError, customError.HTTPCode)
	}
	return web.EncodeJSON(w, model.BusinessError{
		Msg:      "unexpected error",
		HTTPCode: http.StatusInternalServerError,
	}, http.StatusInternalServerError)
}
*/
