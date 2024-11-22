package proto

//for rpc
//simplify proto -> go struct
type SendTcp struct {
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	FromUserId   int    `json:"fromUserId"`
	FromUserName string `json:"fromUserName"`
	ToUserId     int    `json:"toUserId"`
	ToUserName   string `json:"toUserName"`
	RoomId       int    `json:"roomId"`
	Op           int    `json:"op"`
	CreatTime    string `json:"createTime"`
	AuthToken    string `json:"authToken"` // for tcp
}
type ConncetRequest struct {
	AuthToken string `json:"authToken"`
	RoomId    int    `json:"roomId"`
	ServerId  string `json:"serverId"`
}
type ConncetResponse struct {
	UserId int
}
type DisConnectRequest struct {
	RoomId int
	UserId int
}
type DisConnectResponse struct {
	Has bool
}
type Send struct {
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	FromUserId   int    `json:"fromUserId"`
	FromUserName string `json:"fromUserName"`
	ToUserId     int    `json:"toUserId"`
	ToUserName   string `json:"toUserName"`
	RoomId       int    `json:"roomId"`
	Op           int    `json:"op"`
	CreatTime    string `json:"createTime"`
}
type GetUserInfoRequest struct {
	UserId int
}
type GetUserInfoResponse struct {
	Code     int
	UserId   int
	UserName string
}
type LogoutRequest struct {
	AuthToken string
}
type LogoutReponse struct {
	Code int
}

type RegisterRequest struct {
	Name     string
	PassWord string
}
type RegisterResponse struct {
	Code      int
	AuthToken string
}

type LoginRequest struct {
	Name     string
	PassWord string
}
type LoginResponse struct {
	Code      int
	AuthToken string
}

type CheckAuthRequest struct {
	AuthToken string
}
type CheckAuthResponse struct {
	Code     int
	UserId   int
	UserName string
}
