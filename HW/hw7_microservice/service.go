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
	"sync"
	"time"
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

	streams := make([]*Stream, 0, 4)
	mu := &sync.RWMutex{}
	i := Interceptor{
		Alc:     alcMap,
		Streams: &streams,
		mu:      mu,
	}

	server := grpc.NewServer(
		i.UnaryInterceptor(),
		i.StreamInterceptor(),
	)

	bs := &BizServerImpl{}
	RegisterBizServer(server, bs)

	as := &AdminServerImpl{
		Streams: &streams,
		mu:      mu,
	}
	RegisterAdminServer(server, as)
	fmt.Printf("starting server at %s\n", addr)

	go func(ctx context.Context) {
		go server.Serve(lis)
		<-ctx.Done()
		as.Close()
		server.GracefulStop()

	}(ctx)

	return nil
}

type Interceptor struct {
	Alc     map[string][]string
	Streams *[]*Stream
	mu      *sync.RWMutex
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

func (i Interceptor) UnaryInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if err := i.intercept(ctx, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	})
}

func (i Interceptor) StreamInterceptor() grpc.ServerOption {
	return grpc.StreamInterceptor(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := i.intercept(ss.Context(), info.FullMethod); err != nil {
			return err
		}
		return handler(srv, ss)
	})
}

func (i Interceptor) intercept(ctx context.Context, method string) error {
	now := time.Now().Unix()
	md, _ := metadata.FromIncomingContext(ctx)
	if err := checkAuth(i.Alc, md, method); err != nil {
		return err
	}
	i.mu.Lock()
	i.notifyStreams(*i.Streams, &Event{
		Timestamp: now,
		Consumer:  md.Get("consumer")[0],
		Method:    method,
		Host:      "127.0.0.1:",
	})
	i.mu.Unlock()
	return nil
}

func (i Interceptor) notifyStreams(streams []*Stream, ev *Event) {
	for ind, st := range streams {
		select {
		case st.Channel <- ev:
		case <-st.Ctx.Done():
			i.mu.Lock()
			stream := (*i.Streams)[ind]
			*i.Streams = append((*i.Streams)[:ind], (*i.Streams)[ind+1:]...)
			close(stream.Channel)
			i.mu.Unlock()
		}
	}
}

func checkAuth(alc map[string][]string, md metadata.MD, method string) error {
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
