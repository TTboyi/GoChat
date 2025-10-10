package middleware

import (
	//"net/http"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers",
    		"Content-Type, Authorization, X-Requested-With, Accept")
  		c.Writer.Header().Set("Access-Control-Expose-Headers", "Authorization")
  		if c.Request.Method == "OPTIONS" {
    		c.AbortWithStatus(200)
    		return
  	}
		c.Next()
	}
}
