package main

import (
	"fmt"
	"strings"
	"time"
)

//抽象读取、写入接口
type Reader interface {
	Read(chan string) //读方法
}

type Writer interface {
	Write(chan string)
}

type LogProcess struct {
	rc    chan string
	wc    chan string
	read  Reader
	write Writer
}

type ReadFromFile struct {
	path string //读取文件路径
}

func (r *ReadFromFile) Read(rc chan string) {
	//读取模块
	line := "message"
	rc <- line
}

type WriteToInfluxDB struct {
	influxdbDsn string //influxdb的Dsn
}

func (w *WriteToInfluxDB) Write(wc chan string) {
	//写入模块
	fmt.Println(<-wc)
}

func (l *LogProcess) LogParse() {
	//解析模块
	data := <-l.rc
	l.wc <- strings.ToUpper(data)
}

func main() {
	r := &ReadFromFile{
		path: "./access.log",
	}
	w := &WriteToInfluxDB{}
	lp := &LogProcess{
		rc:    make(chan string),
		wc:    make(chan string),
		read:  r,
		write: w,
	}
	go lp.read.Read(lp.rc)
	go lp.LogParse()
	go lp.write.Write(lp.wc)

	time.Sleep(time.Second * 1)
}
