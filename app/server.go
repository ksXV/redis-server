package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
    conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

    data := make([]byte, 100)
    for {
        _, err = conn.Read(data)
        if err != nil {
            if err.Error() == "EOF" {
                err = conn.Close();
                if err != nil {
                    fmt.Println("Error closing the connectiong: ", err.Error())
                    os.Exit(1)
                }
                fmt.Println("Closing conn")
                break;
            }
            fmt.Println("Error reading bytes: ", err.Error())
            os.Exit(1)
        }

        if data[0] != 0 {
            conn.Write([]byte("+PONG\r\n"))
        }
    }

}
