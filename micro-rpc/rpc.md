# 微服务基础--rpc

## rpc简介
  [直接引用维基百科的描述](https://zh.wikipedia.org/wiki/%E9%81%A0%E7%A8%8B%E9%81%8E%E7%A8%8B%E8%AA%BF%E7%94%A8) ：在分布式计算，远程过程调用（英语：Remote Procedure Call，缩写为 RPC）是一个计算机通信协议。该协议允许运行于一台计算机的程序调用另一个地址空间（通常为一个开放网络的一台计算机）的子程序，而程序员就像调用本地程序一样，无需额外地为这个交互作用编程（无需关注细节）。RPC是一种服务器-客户端（Client/Server）模式，经典实现是一个通过发送请求-接受回应进行信息交互的系统 <br>
用途很明确，就是让开发者调用另外一个程序的方法就像是调用本地方法一样简单，这也是rpc为什么会成为微服务的基础的原因，微服务的核心就在于不同服务（不同程序）可以互相调用

## rpc简单示例

因为是client-server模式，所以我们先创建两个文件, client.go  server.go <br>
我们使用go自带的rpc库，在 net/rpc下 <br>
server.go 中：
```go
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
    conn, err := listen.Accept()
    if err != nil {
        log.Fatalf("accept err:%+v", err)
    }
    log.Printf("accept conn:%s \n",  conn.RemoteAddr().String())
    rpc.ServeConn(conn)

}

```

server 中我们定义了echoservice这样一个服务，这个服务下有Hello 方法，Hello 方法有一个入参HelloReq，一个出参HelloRsp，其中HelloReq 等同与http api的请求参数， HelloRsp 等同于http api的响应数据

client.go :
```go
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

```

client中 我们通过拨号访问server中的ip:port ，通过call server中申明的服务名rpc.echoservice 加上方法名Hello 组成的名称rpc.echoservice.Hello，然后传入Hello函数的参数，就像本地有个Hello方法一样

然后在开启两个终端，其中一个终端先启动server
```shell
go run server.go
```
再启动client
```shell
go run client.go
# 2021/11/13 10:40:37 rpc reply:&{Reply:hello Lee!}
```
我们会发现运行一次之后server就退出了，这显然不符合一个服务可以被重复调用的基础能力，是因为我们在server中，accept 了一个请求处理完后就退出了，只要将server接收的地方不停的接收，然后起goroutine去处理请求，server代码改成：
```go
for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalf("accept err:%+v", err)
		}
		log.Printf("accept conn:%s \n",  conn.RemoteAddr().String())
		go rpc.ServeConn(conn)
	}
```

我们再次运行例子，就可以无限的，并发的调用server了

如果让我们来开发这样的一个功能，我们会怎么实现呢
不会不要紧，我们直接学习官方代码的实现即可，go的进阶最简单的路径就是看官方代码，看看大佬们是怎么实现的
我本地go版本是go version go1.16.7 darwin/amd64, 代码在 net/rpc 下，文件非常少，所以实现并不复杂，我们打开server.go
找到RegisterName方法，因为我们刚才自己实现的例子中，main方法的第一行就是rpc.RegisterName("rpc.echoservice", new(EchoService))，
RegisterName 方法中什么都没有，真正的实现在 server.register方法中, 看看注册方法具体做了什么

```go
type Server struct {
    serviceMap sync.Map   // 存服务名和服务的对应的结构体反射信息
    reqLock    sync.Mutex // protects freeReq
    freeReq    *Request
    respLock   sync.Mutex // protects freeResp
    freeResp   *Response
}

type service struct {
    name   string                 // name of service 
    rcvr   reflect.Value          // receiver of methods for the service
    typ    reflect.Type           // type of the receiver
    method map[string]*methodType // registered methods 存方法名和方法的反射信息
}

func (server *Server) register(rcvr interface{}, name string, useName bool) error {
	
	s := new(service) //初始化了一个service的结构体, 这个结构体中包含了EchoService的反射信息和方法
	//rcvr 是 EchoService 的实例的指针，下面两个行是将EchoService的的反射信息存在service结构体中
	s.typ = reflect.TypeOf(rcvr) 
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name() // sname = "EchoService"
	if useName { // userName true
		sname = name  //优先使用传入的name，否则使用反射获取的结构体的名称
	}
	//sname="rpc.echoservice"
	if sname == "" {
		s := "rpc.Register: no service name for type " + s.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	//判断结构题是否暴露出去
	if !token.IsExported(sname) && !useName {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}
	s.name = sname

	// 返回满足rpc的方法，阅读suitableMethods方法的实现可以看出来，需满足如下条件
	//1: 方法必须暴露出去，也就是方法名首字母大写, Hello 方法满足
	//2: 参数的数量必须是3个，包含接受者，请求参数，响应参数 Hello方法对应这三个参数s *EchoService, req HelloReq, rsp *HelloRsp
    //3: 函数的第一个参数必须不是指针，对应的是req HelloReq，因为请求参数都是只读的
    //4： 函数的第二个参数必须是指针，对应的是rsp *HelloRsp，因为响应数据必须是可写的
    //5： servie的必须包含一个暴露的公共方法
    //6：函数的返回必须是个error类型
    // methods := make(map[string]*methodType)
    //methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}，存储了方法的一些反射数据
    
	s.method = suitableMethods(s.typ, true)

	if len(s.method) == 0 {
		str := ""

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(s.typ), false)
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}
    //server.serviceMap 存储了服务对应的方法集合 ,类似用map json表示的话 {"rpc.echoservice":{"Hello":"Hello的反射信息"}}
	if _, dup := server.serviceMap.LoadOrStore(sname, s); dup {
		return errors.New("rpc: service already defined: " + sname)
	}
	return nil
}
```

RegisterName方式实际上就是将服务名rpc.echoservice 存储在map中，map中包含各种方法的反射信息，因为调用的时候需要用到这些反射信息
server中下面的代码就是listen一个网络端口，指定网络类型，然后accpet等待网络连接，等到一个网络请求的到达时，将网络请求传给rpc.ServeConn(conn)方法
我们看下ServeConn如何实现的，具体实现在(server *Server) ServeConn 中，ServeConn 这个方法也比较简单，构造了gobServerCodec 对象，然后调用了ServeCodec方法，接下来看看ServeCodec的实现逻辑
```go

type gobServerCodec struct {
	rwc    io.ReadWriteCloser //当前的连接
	dec    *gob.Decoder //解析请求数据的
	enc    *gob.Encoder //序列化响应数据的
	encBuf *bufio.Writer //实现了conn的写入
	closed bool
}
func (server *Server) ServeCodec(codec ServerCodec) {
    sending := new(sync.Mutex)
    wg := new(sync.WaitGroup)
    for {
    	//拿到service的对应信息，请求参数，响应数据，方法的反射信息，请求参数已经通过codec解析到argv中
        service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
        if err != nil {
            if debugLog && err != io.EOF {
                log.Println("rpc:", err)
            }
            if !keepReading {
                break
            }
            // send a response if we actually managed to read a header.
            if req != nil {
            	//调用
                server.sendResponse(sending, req, invalidRequest, codec, err.Error())
                server.freeRequest(req)
            }
            continue
        }
        wg.Add(1)
        //根据反射调用对应的方法，在call方法中调用server.sendRespons将response 通过gobServerCodec 的WriteResponse序列化后写入到conn中
        go service.call(server, sending, wg, mtype, req, argv, replyv, codec)
    }
    // We've seen that there are no more requests.
    // Wait for responses to be sent before closing codec.
    wg.Wait()
    codec.Close()
}



func (server *Server) readRequestHeader(codec ServerCodec) (svc *service, mtype *methodType, req *Request, keepReading bool, err error) {
	// Grab the request header.
	//从空闲的request的列表获取空闲的request对象
	req = server.getRequest()
	//从请求中解析出请求的服务和方法，具体如何解析后面再分析，serviceMehtod=rpc.echoservice.Hello
	err = codec.ReadRequestHeader(req)
	if err != nil {
		req = nil
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		err = errors.New("rpc: server cannot decode request: " + err.Error())
		return
	}

	// We read the header successfully. If we see an error now,
	// we can still recover and move on to the next request.
	keepReading = true

	dot := strings.LastIndex(req.ServiceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc: service/method request ill-formed: " + req.ServiceMethod)
		return
	}
	serviceName := req.ServiceMethod[:dot]
	methodName := req.ServiceMethod[dot+1:]

	// Look up the request.
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc: can't find service " + req.ServiceMethod)
		return
	}
	//从RegisterName注册的两个map中读取对应的的service和method的的反射信息
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc: can't find method " + req.ServiceMethod)
	}
	return
}
```

到此从接收到网络请求到写回到网络就完成了，除了gobServerCodec的方法没有看，server的主流程就清除了，
codec提供了序列化、反序列化等功能，如果理解的简单一些，可以理解成json的marshal和unmarshal，
codec可以拿来单独分析，暂时不分析不影响阅读rpc的主要逻辑

继续看下client.go如何发起调用，通过server的代码，我们大概能猜到客户端的代码是传入service+method方法的字符串，
然后请求数据通过同样的gobServerCodec发送到网络中，我们带着猜测取看下具体的实现
通过client.Call("rpc.echoservice.Hello", req, rsp)调用Call方法，
call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done 发送请求，等待响应的返回
client.Go()方法中调用client.send(call)，发送请求，client.send()方法调用 client.codec.WriteRequest(&client.request, call.Args)发送请求
发送看起来也比较简答
```go

func (c *gobClientCodec) WriteRequest(r *Request, body interface{}) (err error) {
	//序列化请求头
	if err = c.enc.Encode(r); err != nil {
		return
	}
	//序列化和写入请求参数
	if err = c.enc.Encode(body); err != nil {
		return
	}
	return c.encBuf.Flush()
}
```

最后通过一张图表示整个流程

![rpc](./rpc.png ''rpc'')