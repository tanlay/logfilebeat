package main

import (
	"bufio"
	"context"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
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

type WriteToInfluxDB struct {
	influxdbStr   string
	influxdbToken string
}

type Message struct {
	TimeLocal                    time.Time
	BytesSent                    int
	Path, Method, Scheme, Status string
	UpstreamTime, RequestTime    float64
}

// Read 读取模块
/*
1. 打开文件
2. 从文件末尾开始逐行读取
3. 写入read channel
*/
func (r *ReadFromFile) Read(rc chan []byte) {
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

// LogParse 解析模块
/*
1. 从read channel中读取每行的日志数据
2. 正则匹配日志内容，提取出所需要的监控数据
3. 解析后的数据写入write channel中，供后续写入模块使用
*/
func (l *LogProcess) LogParse() {
	/**
	172.0.0.12 - - [04/Mar/2018:13:49:52 +0000] http "GET /foo?query=t HTTP/1.0" 200 2133 "-" "KeepAliveClient" "-" 1.005 1.854
	*/

	regStr := fmt.Sprintf("%s", `([\d\.]+)\s+([^ \[]+)\s+([^ \[]+)\s+\[([^\]]+)\]\s+([a-z]+)\s+\"([^"]+)\"\s+(\d{3})\s+(\d+)\s+\"([^"]+)\"\s+\"(.*?)\"\s+\"([\d\.-]+)\"\s+([\d\.-]+)\s+([\d\.-]+)`)

	r := regexp.MustCompile(regStr) //编译正则表达式

	loc, _ := time.LoadLocation("Asia/Shanghai")

	for v := range l.rc { //读取channel的内容
		ret := r.FindStringSubmatch(string(v)) //匹配数据内容括号中的内容，
		if len(ret) != 14 {
			log.Println("FindStringSubmatch fail:", string(v))
			continue //继续进行下一次
		}

		message := &Message{}                                                     //定义message结构体，
		t, err := time.ParseInLocation("02/Jan/2006:15:04:05 +0000", ret[4], loc) //解析string时间为golang的时间类型
		if err != nil {
			log.Println("ParseInLocation fail: ", err.Error(), ret[4])
			continue
		}
		message.TimeLocal = t

		byteSent, _ := strconv.Atoi(ret[8])
		message.BytesSent = byteSent

		//GET /foo?query=t HTTP/1.0
		reqSli := strings.Split(ret[6], " ")
		if len(reqSli) != 3 {
			log.Println("strings.Split fail: ", ret[6])
			continue
		}
		message.Method = reqSli[0]
		u, err := url.Parse(reqSli[1])
		if err != nil {
			log.Println("url parse fail: ", err.Error())
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

// Write 写入模块
/*
1. 初始化influxdb client
2. 从write channel中读取监控数据
3. 构造数据并写入influxdb
*/
func (w *WriteToInfluxDB) Write(wc chan *Message) {

	client := influxdb2.NewClient(w.influxdbStr, w.influxdbToken)

	for v := range wc {
		writeAPI := client.WriteAPIBlocking("my-org", "my-bucket")
		// Create point using full params constructor
		p := influxdb2.NewPoint("nginx_log",
			map[string]string{"Path": v.Path, "Method": v.Method, "Scheme": v.Scheme, "Status": v.Status},
			map[string]interface{}{"UpstreamTime": v.UpstreamTime, "RequestTime": v.RequestTime, "BytesSent": v.BytesSent},
			v.TimeLocal)
		// write point immediately
		writeAPI.WritePoint(context.Background(), p)
		log.Println("write success!!!")
	}
}

func main() {
	r := &ReadFromFile{
		path: "./access.log",
	}
	w := &WriteToInfluxDB{
		influxdbStr:   "localhost:8086",
		influxdbToken: "yVKoffoq89t0iR1wsnc-BVqNRiK9vzOd8OHoGVEwAqxS82kSsIPjk1TByg8XsOScTpqhxBuai6y3oiOJrJ-vvQ==",
	}

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
