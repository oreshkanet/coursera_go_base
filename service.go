package main

// protoc --go_out=plugins=grpc:. *.proto

import (
	"context"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
)

// StartMyMicroservice - Старт микросервиса
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		return err
	}

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
	return nil
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
	return nil, nil
}

func (*bizManager) Add(ctx context.Context, nothin *Nothing) (*Nothing, error) {
	return nil, nil
}

func (*bizManager) Test(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return nil, nil
}

func newBizManager() *bizManager {
	return &bizManager{}
}
