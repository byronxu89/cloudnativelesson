package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var (
	InfoLogger  *log.Logger //普通处理日志
	ErrorLogger *log.Logger //错误处理日志
)

//init 对日志进行初始化
func init() {
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

//http 处理函数
func healthz(w http.ResponseWriter, req *http.Request) {
	//取出每个header 的 key value 信息 转入 response header
	for name, headers := range req.Header {

		joinheadval := strings.Join(headers, ";")
		w.Header().Set(name, joinheadval)

	}
	//寻找VERSION 是否有，添加到header中
	version, ok := os.LookupEnv("VERSION")
	if ok {
		w.Header().Set("VERSION", version)
	} else {
		w.Header().Set("VERSION", "0.0.1")
	}
	clientIP, clientport, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		ErrorLogger.Printf("userip: %q is not IP:port", req.RemoteAddr)
	}

	userIP := net.ParseIP(clientIP)
	if userIP == nil {
		ErrorLogger.Printf("userip: %q is not IP:port", req.RemoteAddr)
	}

	//处理存在proxy 的情况
	forward := req.Header.Get("X-Forwarded-For")
	InfoLogger.Printf("IP: %s, Port: %s, Forwarded for: %s\n", clientIP, clientport, forward)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200"))
}

func main() {

	handler := http.NewServeMux()
	handler.HandleFunc("/healthz", healthz)
	handler.HandleFunc("/healthz/", healthz)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found\n"))
	})

	err := http.ListenAndServe(":8090", handler)
	if err != nil {
		ErrorLogger.Fatalf("Could not start server: %s\n", err.Error())
	}
}
