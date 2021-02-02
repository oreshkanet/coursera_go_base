package main

// protoc --go_out=plugins=grpc:. *.proto

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type serverGRPC struct {
	Server *grpc.Server
	lis    net.Listener
	ACL    map[string][]string
}

/*
type ACL struct {
}
*/

var (
	srvGRPC serverGRPC = serverGRPC{
		Server: grpc.NewServer(),
	}
	ACL                          = make(map[string][]string, 0)
	loggerChannel  chan *Event   = make(chan *Event, 0)
	loggerChannels []chan *Event = make([]chan *Event, 0, 0)
	lisAddr        string        = ""
)

// StartMyMicroservice - Старт микросервиса
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {

	ACL = make(map[string][]string, 0)
	err := json.Unmarshal([]byte(ACLData), &ACL)
	if err != nil {
		return err
	}

	lisAddr = listenAddr + ":8082"

	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		return err
	}

	// В отдельной горутине запускаем чтение из контекста признака закрытия сервера
	go func(_lis net.Listener) {
		<-ctx.Done()
		fmt.Println("stop server at :8082")
		// Закрываем все каналы
		/*
			for _, ch1 := range loggerChannels {
				close(ch1)
			}
			close(loggerChannel)
		*/
		loggerChannels = make([]chan *Event, 0, 0)
		loggerChannel = make(chan *Event, 0)
		_lis.Close()
	}(lis)

	go func() {
		for event := range loggerChannel {
			for _, ch1 := range loggerChannels {
				ch1 <- event

			}
		}
	}()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	RegisterBizServer(server, newBizManager())
	RegisterAdminServer(server, newAdminManager())

	fmt.Println("starting server at :8082")
	go server.Serve(lis)

	return nil

}

func unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	//start := time.Now()
	md, _ := metadata.FromIncomingContext(ctx)

	var consumer string = "unknown"
	consumers := md["consumer"]
	if len(consumers) > 0 {
		consumer = consumers[0]
	}
	methodEnabled := false
	if cons, consExist := ACL[consumer]; consExist {
		for _, pattern := range cons {
			if v, _ := regexp.MatchString(pattern, info.FullMethod); v {
				methodEnabled = true
				break
			}
		}
	}
	if !methodEnabled {
		return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated error")
	}

	//go func() {
	loggerChannel <- &Event{Timestamp: 0, Host: lisAddr, Consumer: consumer, Method: info.FullMethod}
	//fmt.Println(info.FullMethod)
	//}()

	reply, err := handler(ctx, req)

	/*
		fmt.Printf(`--
			after incoming call=%v
			req=%#v
			reply=%#v
			time=%v
			md=%v
			err=%v
			`, info.FullMethod, req, reply, time.Since(start), md, err)
	*/

	return reply, err
}

func streamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	// Call 'handler' to invoke the stream handler before this function returns

	ctx := stream.Context()
	md, _ := metadata.FromIncomingContext(ctx)

	var consumer string = "unknown"
	consumers := md["consumer"]
	if len(consumers) > 0 {
		consumer = consumers[0]
	}
	methodEnabled := false
	if cons, consExist := ACL[consumer]; consExist {
		for _, pattern := range cons {
			if v, _ := regexp.MatchString(pattern, info.FullMethod); v {
				methodEnabled = true
				break
			}
		}
	}
	if !methodEnabled {
		return status.Errorf(codes.Unauthenticated, "Unauthenticated error")
	}

	//go func() {
	loggerChannel <- &Event{Timestamp: 0, Host: lisAddr, Consumer: consumer, Method: info.FullMethod}
	//	fmt.Println(info.FullMethod)
	//}()
	//time.Sleep(2 * time.Millisecond)

	err := handler(srv, stream)

	/*
		fmt.Printf(`--
			after incoming call=%v
			req=%#v
			reply=%#v
			time=%v
			md=%v
			err=%v
			`, info.FullMethod, req, reply, time.Since(start), md, err)
	*/

	return err
}

func writeLogAndStatistics(ctx context.Context, method string) {

	var consumer = "unknown"
	if headers, ok := metadata.FromIncomingContext(ctx); ok {
		consumers := headers["consumer"]
		if len(consumers) > 0 {
			consumer = consumers[0]
		}
	}

	loggerChannel <- &Event{Timestamp: 0, Host: lisAddr, Consumer: consumer, Method: method}
}

/***************************************************************************
* Admin Manager
***************************************************************************/

type adminManager struct {
	mu             sync.RWMutex
	loggerChannels []chan *Event
}

func (*adminManager) Logging(inStream *Nothing, srv Admin_LoggingServer) error {
	//go writeLogAndStatistics(srv.Context(), "/main.Admin/Logging")
	//loggerChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Admin/Logging"}

	logChannel := make(chan *Event, 0)

	/*
		go func(_server *Admin_LoggingServer, _ch1 chan *Event) {
			for out := range _ch1 {
				&_server.Send(out)
				break
			}

				go func(_server Admin_LoggingServer, _ch1 chan *Event) {
					for {
						out := <-c
						_server.Send(out)
						break
					}
				}(server, c)

		}(&outStream, loggerChannel)
	*/

	//var wg sync.WaitGroup
	//wg.Add(1)

	loggerChannels = append(loggerChannels, logChannel)
	//go func() {
	//defer wg.Done()
	for {
		select {
		//case <-(*_srv).Context().Done():
		//close(*_ch)
		//	return //outStream.Context().Err()
		case <-srv.Context().Done():
			fmt.Println(srv.Context().Err().Error())
			return nil
		case s := <-logChannel:
			err := srv.SendMsg(s)
			if err != nil {
				fmt.Println(err.Error())
				return nil
			}
		}
	}
	//}()

	/*
		logChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Admin/Logging"}
		logChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Admin/Logging"}
		logChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Admin/Logging"}
		logChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Admin/Logging"}
	*/

	/*
		statChannel := time.NewTicker(time.Duration(100) * time.Millisecond)
		go func() {

			for {
				select {
				case <-statChannel.C:
					err := srv.SendMsg(&Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Admin/Logging"})
					if err != nil {
						return
					}
					return
				}
			}

		}()
	*/

	/*
		for {
			select {
			case <-srv.Context().Done():
				return srv.Context().Err()
			case s := <-logChannel:
				err := srv.Send(s)
				if err != nil {
					return nil
				}
			}
		}
	*/
	//wg.Wait()

	return nil
}

func (*adminManager) Statistics(statInterval *StatInterval, srv Admin_StatisticsServer) error {
	//writeLogAndStatistics(srv.Context(), "/main.Admin/Statistics")
	//loggerChannel = make(chan *Event, 10)
	//writeLogAndStatistics(ctx, "/main.Admin/Statistics")
	//loggerChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Biz/Logging"}

	statChannel := time.NewTicker(time.Duration(statInterval.IntervalSeconds) * time.Second)
	//go func() {

	for {
		select {
		case <-statChannel.C:
			/*
				err := srv.Send(&Stat{ByMethod: map[string]uint64{},
					ByConsumer: map[string]uint64{}})
				if err != nil {
					return nil
				}
			*/
			return nil
		}
	}

	//}()
	return nil
}

func newAdminManager() *adminManager {
	return &adminManager{
		loggerChannels: make([]chan *Event, 0, 0),
	}
}

/***************************************************************************
* Admin Manager
***************************************************************************/

type bizManager struct{}

func (*bizManager) Check(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	//writeLogAndStatistics(ctx, "/main.Biz/Check")
	//loggerChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: consumer, Method: "main.Biz/Logging"}
	return nothing, nil
}

func (*bizManager) Add(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	//writeLogAndStatistics(ctx, "/main.Biz/Add")
	//loggerChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Biz/Logging"}
	return nothing, nil
}

func (*bizManager) Test(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	//writeLogAndStatistics(ctx, "/main.Biz/Test")
	//loggerChannel <- &Event{Timestamp: 0, Host: "127.0.0.1:", Consumer: "logger", Method: "main.Biz/Logging"}
	return nothing, nil
}

func newBizManager() *bizManager {
	return &bizManager{}
}
