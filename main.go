package main

import (
	"context"
	"fmt"
)

const (
	// какой адрес-порт слушать серверу
	listenAddrMain string = "127.0.0.1:8082"

	// кого по каким методам пускать
	ACLDataMain string = `{
	"logger":    ["/main.Admin/Logging"],
	"stat":      ["/main.Admin/Statistics"],
	"biz_user":  ["/main.Biz/Check", "/main.Biz/Add"],
	"biz_admin": ["/main.Biz/*"]
}`
)

func main() {
	println("usage: go test -v")

	ctx, _ := context.WithCancel(context.Background())
	err := StartMyMicroservice(ctx, listenAddrMain, ACLDataMain)
	if err != nil {
		fmt.Printf("cant start server initial: %v", err)
	}
}
