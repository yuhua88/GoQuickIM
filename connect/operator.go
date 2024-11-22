package connect

import "GoQuickIM/proto"

type Operator interface {
	Connect(conn *proto.ConncetRequest) (int, error)
	DisConnect(disConn *proto.DisConnectRequest) (err error)
}
type DefaultOperator struct {
}

//rpc call logic layer
func (o *DefaultOperator) Connect(conn *proto.ConncetRequest) (uid int, err error) {
	rpcConnect := new(RpcConnect)
	uid, err = rpcConnect.Connect(conn)
	return
}

func (o *DefaultOperator) DisConnect(disConn *proto.DisConnectRequest) (err error) {
	rpcConnect := new(RpcConnect)
	err = rpcConnect.DisConnect(disConn)
	return
}
