package main

import (
	"fmt"
	"strings"
	"time"
)

type LogProcess struct {
	path        string //读取文件路径
	influxdbDsn string //influxdb的Dsn
	rc          chan string
	wc          chan string
}

func (l *LogProcess) ReadFromFile() {
	//读取模块
	line := "message"
	l.rc <- line
}

func (l *LogProcess) LogParse() {
	//解析模块
	data := <-l.rc
	l.wc <- strings.ToUpper(data)
}

func (l *LogProcess) WriteInfluxDB() {
	//写入模块
	fmt.Println(<-l.wc)
}

func main() {
	lp := &LogProcess{
		rc: make(chan string),
		wc: make(chan string),
		path:        "./access.log",
		influxdbDsn: "sadfaf",
	}
	go lp.ReadFromFile()
	go lp.LogParse()
	go lp.WriteInfluxDB()

	time.Sleep(time.Second * 1)
}
