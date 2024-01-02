package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type RedisData struct {
	timeReceived         time.Time
	timeUntilDataExpires time.Duration
	data                 string
}

type RedisServer struct {
	commands           []string
	dataIdx            int
	numberOfArgsParsed int
	dataMap            map[string]RedisData
}

func isLetter(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func parseNumber(text string, i int) int {
	number := 0
	for i < len(text) && isDigit(text[i]) {
		number = number*10 + int(text[i]-'0')
		i++
	}
	return number
}

func parseWord(text string) string {
	var wordBuilder strings.Builder
	i := 0
	for i < len(text) && isLetter(text[i]) {
		wordBuilder.WriteByte(text[i])
		i++
	}
	return wordBuilder.String()
}

func (rs *RedisServer) parseArg(text string) {
	if len(rs.commands) < rs.dataIdx {
		panic("Something fucked up :(")
	}
	var arg string

	if isLetter(text[0]) {
		arg = parseWord(text)
	} else if isDigit(text[0]) {
		arg = fmt.Sprint(parseNumber(text, 0))
	}

	rs.commands[rs.dataIdx] = arg
	rs.numberOfArgsParsed++
}

func (rs *RedisServer) parseInput(text string) {
	if text == "" {
		return
	}

	switch text[0] {
	case '*':
		rs.parseTheNumberOfArgs(text)
	case '$':
		rs.parseTheLengthOfAnArg(text)
	default:
		rs.parseArg(text)
	}
}

func (rs *RedisServer) parseTheNumberOfArgs(text string) {
	numberOfArgs := parseNumber(text, 1)
	rs.commands = make([]string, numberOfArgs)
	rs.dataIdx = -1
	rs.numberOfArgsParsed = 0
}

func (rs *RedisServer) parseTheLengthOfAnArg(text string) {
	_ = parseNumber(text, 1)
	rs.dataIdx++
}

func (rs *RedisServer) handleSet() string {
	rd := RedisData{}
	switch len(rs.commands) {
	case 5:
		rd.timeUntilDataExpires = time.Millisecond * time.Duration(parseNumber(rs.commands[4], 0))
		timeSet := time.Now()
		log.Println("Set at:", timeSet)
		rd.timeReceived = timeSet
		fallthrough
	case 3:
		rd.data = rs.commands[2]
	default:
		return "+ERROR: set [key] [value] optional: [px] [milliseconds]"
	}

	rs.dataMap[rs.commands[1]] = rd

	return "+OK"
}

func (rs *RedisServer) handleGet() string {
	if len(rs.commands) != 2 {
		return "+ERROR: get [key]"
	}
	var result string

	possibleRedisData := rs.dataMap[rs.commands[1]]

	if possibleRedisData.data != "" {
		didResultExpired := (time.Now().Sub(possibleRedisData.timeReceived) - possibleRedisData.timeUntilDataExpires) > 0
		if didResultExpired {
			delete(rs.dataMap, rs.commands[1])
			return "$-1"
		}
		result = fmt.Sprintf("+%s", possibleRedisData.data)
	} else {
		result = "$-1"
	}

	return result
}

func (rs *RedisServer) takeCommands() string {
	switch rs.commands[0] {
	case "echo":
		if len(rs.commands) != 2 {
			return "+ERROR: ECHO [what to echo you dummy]"
		}
		return fmt.Sprintf("+%s", rs.commands[1])
	case "ping":
		return "+PONG"
	case "set":
		return rs.handleSet()
	case "get":
		return rs.handleGet()
	default:
		return "+I don't know what you wanted to run so fuck you."
	}
}

func (rs *RedisServer) writeResponse(conn *net.Conn) {
	if rs.numberOfArgsParsed != len(rs.commands) {
		return
	}

	command := rs.takeCommands()

	_, err := (*conn).Write([]byte(fmt.Sprintf("%s\r\n", command)))
	if err != nil {
		log.Fatal(err.Error())
		return
	}
}

func handleAcceptedConn(conn *net.Conn, rs *RedisServer) {
	scanner := bufio.NewScanner(*conn)
	for scanner.Scan() {
		text := scanner.Bytes()
		rs.parseInput(fmt.Sprintf("%s", text))
		rs.writeResponse(conn)
	}
}

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		go func(conn *net.Conn, err error) {
			rs := new(RedisServer)
			rs.dataMap = make(map[string]RedisData)
			if err != nil {
				fmt.Println("Error accepting connection: ", err.Error())
				os.Exit(1)
			}
			handleAcceptedConn(conn, rs)
		}(&conn, err)
	}

}
