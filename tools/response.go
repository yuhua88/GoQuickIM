package tools

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeUnkownError  = -1
	CodeSuccess      = 0
	CodeSessionError = 40000
	CodeFail         = 1
)

var MsgCodeMap = map[int]string{
	CodeUnkownError:  "unKnow error",
	CodeSuccess:      "success",
	CodeFail:         "fail",
	CodeSessionError: "Session error",
}

func SuccessWithMsg(c *gin.Context, msg interface{}, data interface{}) {
	ResponseWithCode(c, CodeSuccess, msg, data)
}
func FailWithMsg(c *gin.Context, msg interface{}) {
	ResponseWithCode(c, CodeFail, msg, nil)
}
func ResponseWithCode(c *gin.Context, msgCode int, msg interface{}, data interface{}) {
	if msg == nil {
		if val, ok := MsgCodeMap[msgCode]; ok {
			msg = val

		} else {
			msg = MsgCodeMap[-1]
		}
	}
	//Separate HTTP status From service
	c.AbortWithStatusJSON(http.StatusOK, gin.H{
		"code":    msgCode,
		"message": msg,
		"data":    data,
	})

}
