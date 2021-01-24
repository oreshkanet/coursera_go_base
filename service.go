package main

import (
	"context"
	"net"
	"fmt"

	"google.golang.org/grpc"
)

// StartMyMicroservice - Старт микросервиса
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		return err
	}

	server := grpc.NewServer()


	fmt.Println("starting server at :8082")
	server.Serve(lis)

	return nil
}