package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"regexp"
)

func StartMyMicroservice(ctx context.Context, addr string, alc string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		lis.Close()
		return err
	}

	alcMap, err := parseAlc(alc)
	if err != nil {
		lis.Close()
		return err
	}

	server := grpc.NewServer(
		authInterceptor(alcMap)...,
	)

	bs := &BizServerImpl{
		Storage: []*Event{},
	}
	RegisterBizServer(server, bs)
	as := &AdminServerImpl{}
	RegisterAdminServer(server, as)
	fmt.Printf("starting server at %s\n", addr)

	go func(ctx context.Context) {
		fmt.Println("start server")
		go server.Serve(lis)
		<-ctx.Done()
		server.GracefulStop()
		fmt.Println("stop server")

	}(ctx)

	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		matched, err := regexp.Match(a, []byte(e))
		if err == nil && matched {
			return true
		}
	}
	return false
}

func authInterceptor(alc map[string][]string) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			if err := checkAuth(alc, ctx, info.FullMethod); err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}),
		grpc.StreamInterceptor(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			if err := checkAuth(alc, ss.Context(), info.FullMethod); err != nil {
				return err
			}
			return handler(srv, ss)
		}),
	}
}

func checkAuth(alc map[string][]string, ctx context.Context, method string) error {
	md, _ := metadata.FromIncomingContext(ctx)
	consumer := md.Get("consumer")
	if len(consumer) > 0 {
		if methods, ok := alc[consumer[0]]; ok {
			if contains(methods, method) {
				return nil
			}
		}
	}
	return status.Error(codes.Unauthenticated, "")
}

func parseAlc(input string) (map[string][]string, error) {
	alcMap := make(map[string]interface{}, 4)
	if err := json.Unmarshal([]byte(input), &alcMap); err != nil {
		return nil, err
	}
	result := make(map[string][]string, 4)
	for k, v := range alcMap {
		if arr, ok := v.([]interface{}); ok {
			strings := make([]string, len(arr))
			for i := range arr {
				if str, ok := arr[i].(string); ok {
					strings[i] = str
				} else {
					return nil, fmt.Errorf("unable to parse alc")
				}
			}
			result[k] = strings
		} else {
			return nil, fmt.Errorf("unable to parse alc")
		}
	}
	return result, nil
}
