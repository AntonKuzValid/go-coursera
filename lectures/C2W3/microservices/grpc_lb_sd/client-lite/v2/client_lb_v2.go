package main

import (
	"context"
	"flag"
	"fmt"
	"go-coursera/lectures/C2W3/microservices/grpc/session"
	"google.golang.org/grpc/balancer/roundrobin"
	"log"
	"strconv"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
)

var (
	consulAddr = flag.String("addr", "127.0.0.1:8500", "consul addr (8500 in original consul)")
)

var (
	consul *consulapi.Client
)

func main() {
	flag.Parse()
	Init()

	grcpConn, err := grpc.Dial(
		"consul://127.0.0.1:8500/session-api",
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithBalancerName(roundrobin.Name),
	)
	if err != nil {
		log.Fatalf("cant connect to grpc")
	}
	defer grcpConn.Close()

	sessManager := session.NewAuthCheckerClient(grcpConn)

	ctx := context.Background()
	step := 1
	for {
		// проверяем несуществуюущую сессию
		// потому что сейчас между сервисами нет общения
		// получаем загшулку
		sess, err := sessManager.Check(ctx,
			&session.SessionID{
				ID: "not_exist_" + strconv.Itoa(step),
			})
		fmt.Println("get sess", step, sess, err)

		time.Sleep(1500 * time.Millisecond)
		step++
	}
}
