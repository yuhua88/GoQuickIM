package handler

import (
	"GoQuickIM/api/rpc"
	"GoQuickIM/proto"
	"GoQuickIM/tools"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type FormLogin struct {
	UserName string `form:"userName" json:"userName" binding:"required"`
	PassWord string `form:"passWord" json:"passWord" binding:"required"`
}

func Login(c *gin.Context) {
	var formLogin FormLogin
	if err := c.ShouldBindBodyWith(&formLogin, binding.JSON); err != nil {
		tools.FailWithMsg(c, err.Error())
		return
	}
	req := &proto.LoginRequest{
		Name:     formLogin.UserName,
		PassWord: formLogin.PassWord,
	}
	code, authToken, msg := rpc.RpcLogicObj.Login(req)
	if code == tools.CodeFail || authToken == "" {
		tools.FailWithMsg(c, msg)
		return
	}
	tools.SuccessWithMsg(c, "login success", authToken)

}

type FormRegister struct {
	UserName string `form:"userName" json:"userName" binding:"required"`
	PassWord string `form:"passWord" json:"passWord" binding:"required"`
}

func Register(c *gin.Context) {
	var formRegister FormRegister
	if err := c.ShouldBindBodyWith(&formRegister, binding.JSON); err != nil {
		tools.FailWithMsg(c, err.Error())
		return
	}
	req := &proto.RegisterRequest{
		Name:     formRegister.UserName,
		PassWord: tools.Sha1(formRegister.PassWord),
	}
	code, authToken, msg := rpc.RpcLogicObj.Register(req)
	if code == tools.CodeFail || msg == "" {
		tools.FailWithMsg(c, msg)
		return
	}
	tools.SuccessWithMsg(c, "register success", authToken)
}

type FormCheckAuth struct {
	AuthToken string `form:"authToken" json:"authToken" binding:"required"`
}

func CheckAuth(c *gin.Context) {
	var formCheckAuth FormCheckAuth
	if err := c.ShouldBindBodyWith(&formCheckAuth, binding.JSON); err != nil {
		tools.FailWithMsg(c, err.Error())
		return
	}
	authToken := formCheckAuth.AuthToken
	req := &proto.CheckAuthRequest{
		AuthToken: authToken,
	}
	code, userId, userName := rpc.RpcLogicObj.CheckAuth(req)
	if code == tools.CodeFail || userId == 0 || userName == "" {
		tools.FailWithMsg(c, "auth fail")
		return
	}
	var jsonData = map[string]interface{}{
		"userId":   userId,
		"userName": userName,
	}
	tools.SuccessWithMsg(c, "auth success", jsonData)
}

type FormLogout struct {
	AuthToken string `form:"authToken" json:"authToken" binding:"required"`
}

func Logout(c *gin.Context) {
	var formLogout FormLogout
	if err := c.ShouldBindBodyWith(&formLogout, binding.JSON); err != nil {
		tools.FailWithMsg(c, err.Error())
		return
	}
	authToken := formLogout.AuthToken
	req := &proto.LogoutRequest{
		AuthToken: authToken,
	}
	code, msg := rpc.RpcLogicObj.Logout(req)
	if code == tools.CodeFail {
		tools.FailWithMsg(c, msg)
		return
	}
	tools.SuccessWithMsg(c, "logout success", nil)
}
