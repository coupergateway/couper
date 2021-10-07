package test

import (
	"net"
	"strconv"
	"time"
)

func WaitForPort(port int) {
	round := time.Duration(0)

	for {
		round++

		conn, dialErr := net.Dial("tcp4", ":"+strconv.Itoa(port))
		if dialErr != nil {
			break
		}
		_ = conn.Close()

		time.Sleep(time.Second + (time.Second*round)/2)

		if round == 20 {
			panic("port is still in use")
		}
	}
}
