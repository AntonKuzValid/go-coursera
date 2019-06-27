package main

import "context"

func main() {
	println("usage: go test -v")

	ACLData := `{
	"logger":    ["/main.Admin/Logging"],
	"stat":      ["/main.Admin/Statistics"],
	"biz_user":  ["/main.Biz/Check", "/main.Biz/Add"],
	"biz_admin": ["/main.Biz/*"]
}`

	listenAddr := "127.0.0.1:8082"
	print(StartMyMicroservice(context.Background(), listenAddr, ACLData))
}
