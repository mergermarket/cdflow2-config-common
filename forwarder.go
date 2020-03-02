package common

import (
	"io"
	"log"
	"net"
	"time"
)

func forward(readStream io.Reader, writeStream io.Writer, socketPath string) {
	connection := connect(socketPath)
	defer connection.Close()
	go func() {
		_, err := io.Copy(connection, readStream)
		if err != nil {
			log.Panicf("error copying to %v: %v", socketPath, err)
		}
		if err := connection.CloseWrite(); err != nil {
			log.Panicln("error closing socket:", err)
		}
	}()
	if _, err := io.Copy(writeStream, connection); err != nil {
		log.Panicf("error copying from %v: %v", socketPath, err)
	}
}

func connect(socketPath string) *net.UnixConn {
	var err error
	for i := 0; i < 20; i++ {
		var connection net.Conn
		connection, err = net.Dial("unix", socketPath)
		if err == nil {
			if connection, ok := connection.(*net.UnixConn); ok {
				return connection
			}
			log.Panicf("unexpected type for connection: %T", connection)
		}
		time.Sleep(200 * time.Millisecond)
	}
	log.Panicf("could not connect to unix domain socket %v after four seconds: %v\n", socketPath, err)
	return nil
}
