package qalarm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"
)

type qalarm struct {
	pid        int    // 项目id
	mid        int    // 模块id
	code       int    // 错误码
	count      int    // 错误次数
	message    string // 错误详细信息
	serverName string // 服务器hostname
	clientIp   string // 客户端ip
	script     string // 错误脚本名字
	countType  string // 错误类型  "inc"  "set"
	time       int64  // 时间
	debug      bool
}

type messageCount struct {
	c  int    `json:"c"` // 项目id
	m  string `json:"m"`
	t  int64  `json:"t"` // 时间
	k  string `json:"k"`
	ip string `json:"ip"`
	ty string `json:"ty"`
	v  string `json:"v"`
}

const (
	log_dir       = "/home/q/php/Qalarm/logs/"
	agent_log_dir = "/home/q/php/Qalarm/logs/wonderagent/"
	c_file        = "qalarm.c." // 次数文件
	m_file        = "qalarm.m." // 详细错误文件
	version       = "3.0"
	perm          = 0777
)

func getMapIntVal(params map[string]interface{}, key string, defaultVal int) (mapval int) {
	val, ok := params[key]
	mapval = defaultVal
	if ok {
		intval, ok := val.(int)
		if ok {
			mapval = intval
		}
	}

	return
}

func getMapStringVal(params map[string]interface{}, key string, defaultVal string) (mapval string) {
	val, ok := params[key]
	mapval = defaultVal
	if ok {
		stringval, ok := val.(string)
		if ok {
			mapval = stringval
		}
	}

	return
}
func getMapBoolVal(params map[string]interface{}, key string, defaultVal bool) (mapval bool) {
	val, ok := params[key]
	mapval = defaultVal
	if ok {
		boolVal, ok := val.(bool)
		if ok {
			mapval = boolVal
		}
	}

	return
}

/**
	构造函数
	NewQalarm(9,1,111,"error message",map[string]interface{}{"pid":1,"mid":3,"code":1232,"message:"我是错误信息"}).Send()
*/
func NewQalarm(pid, mid, code int, message string, params ...map[string]interface{}) *qalarm {
	mergeParams := map[string]interface{}{}
	if len(params) > 0 {
		for _, pOne := range params {
			for key, val := range pOne {
				mergeParams[key] = val
			}
		}
	}
	count := getMapIntVal(mergeParams, "count", 1)
	serverName := getMapStringVal(mergeParams, "serverName", "")
	clientIp := getMapStringVal(mergeParams, "clientIp", "127.0.0.1")
	script := getMapStringVal(mergeParams, "script", "")
	countType := getMapStringVal(mergeParams, "countType", "inc")
	debug := getMapBoolVal(mergeParams, "debug", false)
	if len(serverName) == 0 {
		serverName, _ = os.Hostname()
	}
	if len(script) == 0 {
		_, script2, _, _ := runtime.Caller(1)
		script = script2
	}
	return &qalarm{pid: pid, mid: mid, code: code, count: count, message: message, serverName: serverName, clientIp: clientIp, script: script, countType: countType, debug: debug}
}

/**
	写日志进程
*/
func (this *qalarm) Send() (bool, error) {
	this.println("send 01")

	if this.valid() == false {
		return false, errors.New("pid mid code 和message为必填项")
	}
	this.println("send 02 验证结束")
	this.println("message:", this.message)

	ts := time.Now().Unix()
	this.time = ts
	key := fmt.Sprintf("%d/%d/%d", this.pid, this.mid, this.code)
	path := fmt.Sprintf("%d/%d", this.pid, this.mid)
	fileName := fmt.Sprintf("/%d", this.code)
	rs := this.readFile(key)
	this.println("rs:", rs)

	var msg messageCount
	json.Unmarshal([]byte(rs), &msg)
	sync_m := true
	if len(rs) > 0 {
		this.println("msg:", msg.c, msg.t, msg.ip, msg.k, msg.m)
		diff := ts - msg.t
		this.println("diff:", diff)
		this.println("diff:", diff)
		if diff >= 1 && diff <= 5 {
			msgc := map[string]interface{}{"c": this.count, "t": this.time, "k": key, "ip": this.serverName, "m": this.message, "ty": this.countType, "v": version}
			con, err := json.Marshal(msgc)
			if err != nil {
				this.println("json失败:", err)
				return false, err
			}
			this.println("msgc1:json之后的内容是 ", string(con))
			this.writeMsg(path, fileName, string(con))
			this.writeLog(c_file, "\n" + rs)
		} else if diff > 5 {
			msgc := map[string]interface{}{"c": this.count, "t": this.time, "k": key, "ip": this.serverName, "m": this.message, "ty": this.countType, "v": version}
			con, err := json.Marshal(msgc)
			if err != nil {
				this.println("json失败:", err)
				return false, err
			}
			this.println("msgc1:json之后的内容是 ", string(con))
			this.writeMsg(path, fileName, string(con))
			this.writeLog(c_file, string(con) + "\n")
		} else {
			count := this.count
			//this.println("1 msg.c:", msg.C, " this.count:", this.Count)
			if this.countType != "set" {
				count = this.count + msg.c
				//this.println("2 msg.c:", msg.C, " this.count:", this.Count, )

			}
			//this.println("3 msg.c:", msg.C, " this.count:", this.Count, " count:", count)
			msgc := map[string]interface{}{"c": count, "t": this.time, "k": key, "ip": this.serverName, "m": this.message, "ty": this.countType, "v": version}
			con, err := json.Marshal(msgc)
			if err != nil {
				this.println("json失败:", err)
				return false, err
			}
			this.println("msgc1:json之后的内容是 ", string(con))
			this.writeMsg(path, fileName, string(con))

			if this.countType != "set" && count > 10 {
				sync_m = false
			}

		}

	} else if len(rs) == 0 {
		//msgc := &MessageCount{c:this.Count, t:this.Time, k:key, ip:this.ServerName, m:this.Message, ty:this.CountType, v:VERSION}
		msgc := map[string]interface{}{"c": this.count, "t": this.time, "k": key, "ip": this.serverName, "m": this.message, "ty": this.countType, "v": version}
		con, err := json.Marshal(msgc)
		if err != nil {
			this.println("json失败:", err)
			return false, err
		}
		this.println("msgc:json之后的内容是 ", string(con))
		this.writeMsg(path, fileName, string(con))
		this.writeLog(c_file, string(con) + "\n")

	} else {
		msgc := map[string]interface{}{"c": this.count, "t": this.time, "k": key, "ip": this.serverName, "ty": this.countType, "v": version}
		con, err := json.Marshal(msgc)
		if err != nil {
			this.println("json失败:", err)
			return false, err
		}
		this.println("msgc:json之后的内容是 ", string(con))
		this.writeMsg(path, fileName, string(con))
		sync_m = false
	}

	this.message = strings.Replace(this.message, "\n", "<br/>", -1)
	this.script = strings.Replace(this.script, "\n", "<br/>", -1)

	msgall := map[string]interface{}{"time": time.Now().Format("2006-01-02 15:04:05"), "pid": this.pid, "mid": this.mid, "code": this.code, "message": this.message, "server_ip": this.serverName, "client_ip": this.clientIp, "script": this.script}
	conall, err := json.Marshal(msgall)
	if err != nil {
		this.println("json失败:", err)
		return false, err
	}
	this.writeAllLog("\n" + string(conall))
	if sync_m {

		if len(string(conall)) > 2020 {
			this.message = this.message[:1800]
			this.script = this.script[:90]
			msgall = map[string]interface{}{"time": time.Now().Format("2006-01-02 15:04:05"), "pid": this.pid, "mid": this.mid, "code": this.code, "message": this.message, "server_ip": this.serverName, "client_ip": this.clientIp, "script": this.script}
			conall, err = json.Marshal(msgall)
			if err != nil {
				this.println("json失败:", err)
				return false, err
			}
		}
		this.writeLog(m_file, string(conall) + "\n")
	}
	return true, nil
}

/**
	取得当前日期
*/
func (this *qalarm) getToday() string {
	return time.Now().Format("20060102")
}

/**
	写 qlarm_c  qalarm_m
*/
func (this *qalarm) writeLog(logType, content string) (bool, error) {
	return this.writeFile(agent_log_dir + logType + this.getToday(), content, true)
}

/*
	写qalarm总的日志
*/
func (this *qalarm) writeAllLog(content string) (bool, error) {
	return this.writeFile(agent_log_dir + "alarm.log." + this.getToday(), content, true)
}

/**
	写qalarm每个错误就是合并的那个
*/
func (this *qalarm) writeMsg(path, fileName, content string) (bool, error) {
	filePath := log_dir + path
	fileExists, err := this.pathExists(filePath)
	if err != nil {
		return false, err
	}
	if !fileExists {
		err := os.MkdirAll(filePath, perm)
		if err != nil {
			return false, err
		}
	}
	file := filePath + "/" + fileName
	return this.writeFile(file, content, false)
}

func (this *qalarm) pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

/**
	追加和覆盖
*/
func (this *qalarm) writeFile(file, content string, append bool) (issuccess bool, err error) {
	var fileout *os.File
	if !append {
		fileout, err = os.OpenFile(file, os.O_WRONLY | os.O_TRUNC | os.O_CREATE, os.ModePerm)
		defer fileout.Close()
	} else {
		fileout, err = os.OpenFile(file, os.O_APPEND | os.O_WRONLY, os.ModePerm)
		defer fileout.Close()
	}

	if err != nil {
		this.println("打开文件失败:", file, "error:", err.Error())
		if _, err := os.Stat("/path/to/whatever"); os.IsNotExist(err) {
			fileout, err = os.Create(file)
			defer fileout.Close()
		}
	}
	this.println("文件:", file, "写入内容:", content)
	_, err = fileout.Write([]byte(content))
	if err != nil {
		this.println("写入错误", err.Error())
	}
	return err == nil, err
}

/*
	只读
*/
func (this *qalarm) readFile(file string) string {
	path := log_dir + file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(content)
}

/**
	验证   pid  code  mid 和 message是否有值
*/
func (this *qalarm) valid() bool {
	if this.pid > 0 && this.code > 0 && this.mid > 0 && len(this.message) > 0 {
		return true
	}
	return false
}

func (this *qalarm) println(arr ...interface{}) {
	if this.debug {
		fmt.Println(arr)
	}
}

//  用法   qalarm.NewQalarm(pit,mid,code,message,map[string]interface{}{"Count":1,"CountType":"inc","ServerName":"dev01.add.sjbs.xxx.com"}).Send()

//func main() {
//times := 1
//if len(os.Args) > 1 {
//	times, _ = strconv.Atoi(os.Args[1])
//}
//for i := 0; i < times; i++ {
//	message := "this is go test message" + time.Now().Format("2006-01-02 15:04:05")
//	NewQalarm(97, 1, 668, message, map[string]interface{}{}).Send()
//}
//}