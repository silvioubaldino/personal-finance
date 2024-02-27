package _import

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"personal-finance/internal/plataform/authentication"
)

type importHandler struct {
	uplannerService Uplanner
}

func NewImportHandlers(r *gin.Engine, uplannersrv Uplanner) {
	handler := importHandler{
		uplannersrv,
	}
	importGroup := r.Group("/import")

	importGroup.POST("/uplanner", handler.ImportUplanner())
}

func (i importHandler) ImportUplanner() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authentication.GetUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err)
			return
		}

		file := c.Request.Body

		//reader := csv.NewReader(file)
		//_, err := reader.ReadAll()
		//if err != nil {
		//	c.JSON(http.StatusInternalServerError, err)
		//	return
		//}

		err = i.uplannerService.Import(file, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
	}
}
