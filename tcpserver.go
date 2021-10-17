package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

func test() {

	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err.Error())
		}
		go handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	request(conn)
	response(conn)
}

func request(conn net.Conn) {
	i := 0
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if i == 0 {
			m := strings.Fields(line)[0]
			fmt.Println("Methods", m)
		}
		if line == "" {
			break
		}
		i++
	}
}

func response(conn net.Conn) {
	body := ``

	fmt.Fprint(conn, "HTTP/1.1 200 OK\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(body))
	fmt.Fprint(conn, "Content-Type: text/html\r\n")
	fmt.Fprint(conn, "\r\n")
	fmt.Fprint(conn, body)
}
