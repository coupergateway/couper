package test

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

const waitForBackoff = 10

func WaitForClosedPort(port int) {
	start := time.Now()
	round := time.Duration(0)

	for {
		conn, dialErr := net.Dial("tcp4", ":"+strconv.Itoa(port))
		if conn != nil {
			_ = conn.Close()
		}
		if dialErr != nil {
			break
		}

		round++
		if round == waitForBackoff {
			panic(fmt.Sprintf("port is still in use after %s: %d", time.Since(start).String(), port))
		}
		time.Sleep(time.Second + (time.Second*round)/2)
	}
}

func WaitForOpenPort(port int) {
	start := time.Now()
	round := time.Duration(0)
	for {
		conn, dialErr := net.Dial("tcp4", ":"+strconv.Itoa(port))
		if conn != nil {
			_ = conn.Close()
		}
		if dialErr == nil {
			return
		}

		time.Sleep(time.Second + (time.Second*round)/2)
		round++
		if round == waitForBackoff {
			panic(fmt.Sprintf("port is still not listening after %s: %d", time.Since(start).String(), port))
		}
	}
}
