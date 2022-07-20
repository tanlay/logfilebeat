package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//抽象读取、写入接口
type Reader interface {
	Read(chan []byte) //读方法
}

type Writer interface {
	Write(chan *Message)
}

type LogProcess struct {
	rc    chan []byte
	wc    chan *Message
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
		if err == io.EOF {              //判断是否是文件结尾
			time.Sleep(500 * time.Millisecond)
			continue
		} else if err != nil {
			panic(fmt.Sprintf("ReadBytes error: %s", err.Error()))
		}
		rc <- line[:len(line)-1] //把读取的内容写入channel中并去掉换行符
	}

}

type WriteToInfluxDB struct {
	influxdbDsn string //influxdb的Dsn
}

type Message struct {
	TimeLocal                    time.Time
	BytesSent                    int
	Path, Method, Scheme, Status string
	UpstreamTime, RequestTime    float64
}

func (w *WriteToInfluxDB) Write(wc chan *Message) {
	//写入模块
	for v := range wc {
		fmt.Println(v)
	}
}

func (l *LogProcess) LogParse() {
	//解析模块
	/**
	172.0.0.12 - - [04/Mar/2018:13:49:52 +0000] http "GET /foo?query=t HTTP/1.0" 200 2133 "-" "KeepAliveClient" "-" 1.005 1.854
	*/

	regStr := fmt.Sprintf("%s", `([\d\.]+)\s+([^ \[]+)\s+([^ \[]+)\s+\[([^\]]+)\]\s+([a-z]+)\s+\"([^"]+)\"\s+(\d{3})\s+(\d+)\s+\"([^"]+)\"\s+\"(.*?)\"\s+\"([\d\.-]+)\"\s+([\d\.-]+)\s+([\d\.-]+)`)

	r := regexp.MustCompile(regStr)	//编译正则表达式

	loc, _ := time.LoadLocation("Asia/Shanghai")

	for v := range l.rc {	//读取channel的内容
		ret := r.FindStringSubmatch(string(v)) //匹配数据内容括号中的内容，
		if len(ret) != 14 {
			log.Println("FindStringSubmatch fail:", string(v))
			continue //继续进行下一次
		}

		message := &Message{}	//定义message结构体，
		t, err := time.ParseInLocation("02/Jan/2006:15:04:05 +0000",ret[4], loc)	//解析string时间为golang的时间类型
		if err != nil{
			log.Println("ParseInLocation fail: ", err.Error(),ret[4])
			continue
		}
		message.TimeLocal = t

		byteSent, _ := strconv.Atoi(ret[8])
		message.BytesSent = byteSent

		//GET /foo?query=t HTTP/1.0
		reqSli := strings.Split(ret[6], " ")
		if len(reqSli) != 3 {
			log.Println("strings.Split fail: ",ret[6])
			continue
		}
		message.Method = reqSli[0]
		u, err := url.Parse(reqSli[1])
		if err != nil{
			log.Println("url parse fail: ",err.Error())
			continue
		}
		message.Path = u.Path
		message.Scheme = ret[5]
		message.Status = ret[7]
		upstreamTime, _ := strconv.ParseFloat(ret[12], 64)
		requestTime, _ := strconv.ParseFloat(ret[13], 64)
		message.UpstreamTime = upstreamTime
		message.RequestTime = requestTime

		l.wc <- message
	}
}

func main() {
	r := &ReadFromFile{
		path: "./access.log",
	}
	w := &WriteToInfluxDB{}
	lp := &LogProcess{
		rc:    make(chan []byte),
		wc:    make(chan *Message),
		read:  r,
		write: w,
	}
	go lp.read.Read(lp.rc)
	go lp.LogParse()
	go lp.write.Write(lp.wc)

	time.Sleep(time.Second * 30)
}
