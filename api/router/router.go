package router

import (
	"GoQuickIM/api/handler"
	"GoQuickIM/api/rpc"
	"GoQuickIM/proto"
	"GoQuickIM/tools"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func Register() *gin.Engine {
	r := gin.Default()
	r.Use(CorsMiddleware())
	//register user push
	initUserRouter(r)
	initPushRouter(r)
	r.NoRoute(func(c *gin.Context) {
		tools.FailWithMsg(c, "please check request url !")
	})
	return r
}
func initPushRouter(r *gin.Engine) {
	pushGroup := r.Group("/push")
	pushGroup.Use(CheckSeesionId())
	{
		pushGroup.POST("/push", handler.Push)
		pushGroup.POST("/pushRoom", handler.PushRoom)
		pushGroup.POST("/count", handler.Count)
		pushGroup.POST("/getRoomInfo", handler.GetRoomInfo)
	}
}
func initUserRouter(r *gin.Engine) {
	userGroup := r.Group("/user")
	userGroup.POST("/login", handler.Login)
	userGroup.POST("/register", handler.Register)
	//check logged in
	userGroup.Use(CheckSeesionId())
	{
		//check auth
		userGroup.POST("/checkAuth", handler.CheckAuth)
		userGroup.POST("/logout", handler.Logout)
	}
}

type FormCheckSessionId struct {
	AuthToken string `form:"authToken" json:"authToken" binding:"required"`
}

func CheckSeesionId() gin.HandlerFunc {
	return func(c *gin.Context) {
		var formCheckSessionId FormCheckSessionId
		if err := c.ShouldBindBodyWith(&formCheckSessionId, binding.JSON); err != nil {
			c.Abort()
			tools.ResponseWithCode(c, tools.CodeSessionError, nil, nil)
			return
		}
		authToken := formCheckSessionId.AuthToken
		req := &proto.CheckAuthRequest{
			AuthToken: authToken,
		}
		code, userId, userName := rpc.RpcLogicObj.CheckAuth(req)
		if code == tools.CodeFail || userId == 0 || userName == "" {
			c.Abort()
			tools.ResponseWithCode(c, tools.CodeSessionError, nil, nil)
			return
		}
		c.Next()
		return
	}
}

// base CORS
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		var openCorsFlag = true
		if openCorsFlag {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Headers", "Origin,X-Requested-With,Content-Type,Accept")
			c.Header("Access-Control-Allow-Methods", "GET,OPTIONS,POST,PUT,DELETE")
			c.Set("content-type", "application/json")
		}
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, nil)
		}
		c.Next()
	}
}
