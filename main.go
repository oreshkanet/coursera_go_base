package main

import (
	"context"
	"fmt"
)

const (
	// какой адрес-порт слушать серверу
	listenAddr string = "127.0.0.1:8082"

	// кого по каким методам пускать
	ACLData string = `{
	"logger":    ["/main.Admin/Logging"],
	"stat":      ["/main.Admin/Statistics"],
	"biz_user":  ["/main.Biz/Check", "/main.Biz/Add"],
	"biz_admin": ["/main.Biz/*"]
}`
)

func main() {
	println("usage: go test -v")

	ctx, _ := context.WithCancel(context.Background())
	err := StartMyMicroservice(ctx, listenAddr, ACLData)
	if err != nil {
		fmt.Printf("cant start server initial: %v", err)
	}
	/*
		wait(1)
		finish() // при вызове этой функции ваш сервер должен остановиться и освободить порт
		wait(1)
	*/
}
