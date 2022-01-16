package main

import (
	"assignment4/metrics"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	InfoLogger  *log.Logger //普通处理日志
	ErrorLogger *log.Logger //错误处理日志
)

type HttpSrv struct {
	Port int

	server    *http.Server
	isStarted bool
	mtx       *sync.Mutex
}

func randInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}

func NewHttpSrv(port int) *HttpSrv {
	return &HttpSrv{
		Port:      port,
		server:    nil,
		isStarted: false,
		mtx:       &sync.Mutex{},
	}
}
func rootHandler(w http.ResponseWriter, r *http.Request) {
	timer := metrics.NewTimer()
	defer timer.ObserveTotal()
	delay := randInt(10, 2000)
	time.Sleep(time.Millisecond * time.Duration(delay))
	//w.Header().Set("Content-Type", "text/plain")
	//w.WriteHeader(http.StatusNotFound)
	//w.Write([]byte("Not found\n"))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hi there!"))
}

func (srv *HttpSrv) Start() (err error) {
	srv.mtx.Lock()
	defer srv.mtx.Unlock()

	if srv.isStarted {
		return errors.New("Server is already started")
	}

	srv.isStarted = true

	metrics.Register()
	// prepare router

	handler := http.NewServeMux()
	handler.HandleFunc("/healthz", healthz)
	handler.HandleFunc("/healthz/", healthz)
	handler.Handle("/metrics", promhttp.Handler())

	handler.HandleFunc("/", rootHandler)

	// prepare address
	addr := fmt.Sprintf(":%v", srv.Port)

	// prepare http server
	srv.server = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	InfoLogger.Printf("Starting http server: %v", srv)

	go func() {
		if err = srv.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				InfoLogger.Printf("Server closed under request: %v", err)
			} else {
				ErrorLogger.Printf("Server closed unexpectedly: %v", err)
			}
		}
		// in case of closed normally
		srv.isStarted = false
	}()

	time.Sleep(10 * time.Millisecond)

	return
}
func (m *HttpSrv) Shutdown(ctx context.Context) (err error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if !m.isStarted || m.server == nil {
		return errors.New("Server is not started")
	}

	stop := make(chan struct{}, 1)
	go func() {
		// dummy preprocess before interrupted
		//time.Sleep(4 * time.Second)

		// Close immediately closes all active net.Listeners and any
		// connections in state StateNew, StateActive, or StateIdle. For a
		// graceful shutdown, use Shutdown.
		//
		// Close does not attempt to close (and does not even know about)
		// any hijacked connections, such as WebSockets.
		//
		// Close returns any error returned from closing the Server's
		// underlying Listener(s).
		//err = m.server.Close()
		// We can use .Shutdown to gracefully shuts down the server without
		// interrupting any active connection
		err = m.server.Shutdown(ctx)
		stop <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		InfoLogger.Printf("Timeout: %v", ctx.Err())
		break
	case <-stop:
		InfoLogger.Printf("Finished")
	}

	return
}

//init 对日志进行初始化
func init() {

	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
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

	srv := NewHttpSrv(8090)
	if err := srv.Start(); err != nil {
		ErrorLogger.Printf("Start failed: %v", err)
		return
	}
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	// holding here
	// waiting for a interrupt signal
	<-quit
	InfoLogger.Printf("[Control-C] Get signal: shutdown server ...")
	signal.Reset(os.Interrupt)

	// starting shutting down progress...
	InfoLogger.Println("Server shutting down")
	// context: wait for 3 seconds
	/*
		ctx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second)
		defer cancel()
	*/
	ctx := context.Background()

	if err := srv.Shutdown(ctx); err != nil {
		ErrorLogger.Printf("Server Shutdown failed: %v", err)
	}
	InfoLogger.Println("Server exiting")
}
