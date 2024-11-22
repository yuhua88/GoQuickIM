package tools

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/bwmarrin/snowflake"
)

const SessionPrefix = "sess_"

func Sha1(s string) (str string) {
	h := sha1.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}
func GetSnowflakeId() string {
	//dafault node id eq 1,this can modify to different serverID node?
	node, _ := snowflake.NewNode(1)
	//Generate a snowflake Id
	id := node.Generate().String()
	return id
}

func GetRandomToken(length int) string {
	r := make([]byte, length)
	//cryptographically secure
	io.ReadFull(rand.Reader, r)
	return base64.URLEncoding.EncodeToString(r)
}

func CreateSessionId(sessionId string) string {
	return SessionPrefix + sessionId
}

func GetSessionIdByUserId(userId int) string {
	return fmt.Sprintf("sess_map_%d", userId)
}

func GetSessionName(sessionId string) string {
	return SessionPrefix + sessionId
}

func GetNowDateTime() string {
	return time.Unix(time.Now().Unix(), 0).Format("2024-11-15 15:04:05")
}
