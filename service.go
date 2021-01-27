package main

// protoc --go_out=plugins=grpc:. *.proto

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type serverGRPC struct {
	Server *grpc.Server
	lis    net.Listener
}

type ACL struct {
}

var (
	srvGRPC serverGRPC = serverGRPC{
		Server: grpc.NewServer(),
	}
)

// StartMyMicroservice - Старт микросервиса
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {

	var ACL = make(map[string][]string, 0)
	err := json.Unmarshal([]byte(ACLData), &ACL)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		return err
	}

	// В отдельной горутине запускаем чтение из контекста признака закрытия сервера
	go func(_lis net.Listener) {
		<-ctx.Done()
		fmt.Println("stop server at :8082")
		_lis.Close()
	}(lis)

	server := grpc.NewServer()

	RegisterAdminServer(server, newAdminManager())
	RegisterBizServer(server, newBizManager())

	fmt.Println("starting server at :8082")
	go server.Serve(lis)

	return nil

}

/***************************************************************************
* Admin Manager
***************************************************************************/

type adminManager struct {
	mu sync.RWMutex
}

func (*adminManager) Logging(nothing *Nothing, server Admin_LoggingServer) error {
	for {
		out := &Event{}
		server.Send(out)
		return nil
	}
}

func (*adminManager) Statistics(statInterval *StatInterval, server Admin_StatisticsServer) error {
	return nil
}

func newAdminManager() *adminManager {
	return &adminManager{}
}

/***************************************************************************
* Admin Manager
***************************************************************************/

type bizManager struct{}

func (*bizManager) Check(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return nothing, nil
}

func (*bizManager) Add(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return nothing, nil
}

func (*bizManager) Test(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return nothing, nil
}

func newBizManager() *bizManager {
	return &bizManager{}
}
