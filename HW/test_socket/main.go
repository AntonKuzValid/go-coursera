package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	_ "syscall"
	"time"
)

//func WriteTo(conn net.Conn, file *os.File, writePos uint32, size uint32) (written uint32, err error) {
//
//	var remain int64 = int64(size)
//	var offset int64 = int64(writePos)
//
//	tcpConn, ok := conn.(*net.TCPConn)
//	if !ok {
//		return 0, errors.New("not a TCPConn")
//	}
//
//	tcpFile, err := tcpConn.File()
//	if err != nil {
//		return 0, err
//	}
//	defer tcpFile.Close()
//	dst := tcpFile.Fd()
//	src := int(file.Fd())
//
//	for remain > 0 {
//		n = int(remain)
//		n, err1 := syscall.Sendfile(dst, src, offset, n)
//		if n > 0 {
//			written += int64(n)
//			remain -= int64(n)
//		}
//		if n == 0 && err1 == nil {
//			break
//		}
//		if err1 == syscall.EAGAIN {
//			continue
//		}
//		if err1 != nil {
//			err = fmt.Errorf("sendfile failed err %s", err1.Error())
//			break
//		}
//	}
//	return written, err
//}

func main() {

	http.HandleFunc(":80", func(writer http.ResponseWriter, request *http.Request) {
		writer.
	})
	newFile, _ := os.Create("new_file")
	newFile.WriteString("hello")
	newFile.Close()
	file, _ := os.Open("new_file")
	//src := int(file.Fd())
	go func(file *os.File) {
		listener, err := net.Listen("tcp", ":8000")
		fmt.Println(err)
		conn, _ := listener.Accept()
		defer conn.Close()
		tcpListener := conn.(*net.TCPConn)
		tcpListener.ReadFrom(file)
		//f, _ := tcpListener.File()
		//dst := int(f.Fd())
		//var offset int64 = 0
		//
		//n, err := syscall.Sendfile(dst, src, &offset, 100)
		//fmt.Println(err)
		//fmt.Println(n)
	}(file)

	start := time.Now().UnixNano()
	conn, err := net.Dial("tcp", ":8000")
	if err == nil {
		buff := make([]byte, 1)
		conn.Read(buff)
		fmt.Println(time.Now().UnixNano() - start)
		fmt.Println(string(buff))
	}
}
