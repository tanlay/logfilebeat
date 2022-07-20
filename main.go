package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

//抽象读取、写入接口
type Reader interface {
	Read(chan []byte) //读方法
}

type Writer interface {
	Write(chan string)
}

type LogProcess struct {
	rc    chan []byte
	wc    chan string
	read  Reader
	write Writer
}

type ReadFromFile struct {
	path string //读取文件路径
}

func (r *ReadFromFile) Read(rc chan []byte) {
	//读取模块

	//1、打开文件
	f, err := os.Open(r.path)
	if err != nil {
		panic(fmt.Sprintf("open file error: %s", err.Error()))
	}
	//2、从文件末尾逐行读取文件内容，避免读取旧的日志文件内容
	f.Seek(0, 2) //字符指针移至末尾
	rd := bufio.NewReader(f)
	for { //循环读取递增的文件内容
		//按行读取文件内容
		line, err := rd.ReadBytes('\n') //读取一行内容，直至换行符结束
		if err == io.EOF {	//判断是否是文件结尾
			time.Sleep(500 * time.Millisecond)
			continue
		} else if err != nil {
			panic(fmt.Sprintf("ReadBytes error: %s", err.Error()))
		}
		rc <- line[:len(line)-1]	//把读取的内容写入channel中并去掉换行符
	}

}

type WriteToInfluxDB struct {
	influxdbDsn string //influxdb的Dsn
}

func (w *WriteToInfluxDB) Write(wc chan string) {
	//写入模块
	for  v:= range wc{
		fmt.Println(v)
	}
}

func (l *LogProcess) LogParse() {
	//解析模块
	for v := range l.rc{
		l.wc <- strings.ToUpper(string(v))
	}
}

func main() {
	r := &ReadFromFile{
		path: "./access.log",
	}
	w := &WriteToInfluxDB{}
	lp := &LogProcess{
		rc:    make(chan []byte),
		wc:    make(chan string),
		read:  r,
		write: w,
	}
	go lp.read.Read(lp.rc)
	go lp.LogParse()
	go lp.write.Write(lp.wc)

	time.Sleep(time.Second * 30)
}
