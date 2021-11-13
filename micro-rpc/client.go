package main

import (
	"log"
	"net/rpc"
)

//请求参数
type HelloReq struct {
	Name string
}

//响应参数
type HelloRsp struct {
	Reply string
}

func main() {
	client, err := rpc.Dial("tcp", "127.0.0.1:6666")
	if err != nil {
		log.Fatalf("rpc dial err:%+v", err)
	}
	req := HelloReq{Name: "Lee"}
	rsp := new(HelloRsp)
	err = client.Call("rpc.echoservice.Hello", req, rsp)
	if err != nil {
		log.Fatalf("rcp call err:%+v", err)
	}
	log.Printf("rpc reply:%+v", rsp)
}
