package main

import (
	"log"
	"net"
	"net/rpc"
)

type EchoService struct {
	//此处可以注入一些服务的依赖，比如db，上游的rpc服务
}

//请求参数
type HelloReq struct {
	Name string
}

//响应参数
type HelloRsp struct {
	Reply string
}

func (s *EchoService) Hello(req HelloReq, rsp *HelloRsp) error {
	rsp.Reply = "hello " + req.Name + "!"
	return nil
}

func main() {
	rpc.RegisterName("rpc.echoservice", new(EchoService))

	listen, err := net.Listen("tcp", ":6666")
	if err != nil {
		log.Fatalf("listen err:%+v", err)
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalf("accept err:%+v", err)
		}
		log.Printf("accept conn:%s \n",  conn.RemoteAddr().String())
		go rpc.ServeConn(conn)
	}

}
