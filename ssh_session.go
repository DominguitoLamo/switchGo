package switchgo

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSH配置
type SSHConfig struct {
	user string
	password string
	ipPort string
	brand string
}

func (config *SSHConfig) GetSessionKey() string {
	return config.user + "_" + config.password + "_" + config.ipPort
}

/**
 * 封装的ssh session，包含原生的ssh.Ssssion及其标准的输入输出管道，同时记录最后的使用时间
 * @attr   session:原生的ssh session，in:绑定了session标准输入的管道，out:绑定了session标准输出的管道，lastUseTime:最后的使用时间
 * @author shenbowei
 */
 type SSHSession struct {
	session     *ssh.Session
	// user + password + ipport
	sessionKey 	string
	in          chan string
	out         chan string
	brand       string
	lastUseTime time.Time
	manager *SessionManager
}

func SSHConfigCreate(user, password, hostname, port, brand string) (*SSHConfig, error) {
	if (user == "" || password == "" || hostname == "" || port == "") {
		return nil, errors.New("config empty")
	}

	if err := ipFormatValid(hostname); err != nil {
		return nil, err
	}

	sshConfig := new(SSHConfig)
	sshConfig.user = user
	sshConfig.password = password
	sshConfig.ipPort = hostname + ":" + port
	sshConfig.brand = brand
	return sshConfig, nil
}

func ipFormatValid(ip string) error {
	numbers := strings.Split(ip, ".")
	if (len(numbers) != 4) {
		return errors.New("This is not IPV4 format!!!!")
	}

	for _, n := range numbers {
		i, err := strconv.Atoi(n)
		if (err != nil) {
			return errors.New("Part of the ip is not number")
		}

		if (i < 0 || i > 255) {
			return errors.New("Part of the ip number is not in the range from 0 to 255")
		}
	}

	return nil
}

/**
 * 创建一个SSHSession，相当于SSHSession的构造函数
 * @param user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口
 * @return 打开的SSHSession，执行的错误
 * @author shenbowei
 */
func NewSSHSession(config *SSHConfig, brand string, manager *SessionManager) (*SSHSession, error) {
	sshSession := new(SSHSession)
	if err := sshSession.createConnection(config); err != nil {
		ErrorLog("NewSSHSession createConnection error:%s", err.Error())
		return nil, err
	}
	if err := sshSession.muxShell(); err != nil {
		ErrorLog("NewSSHSession muxShell error:%s", err.Error())
		return nil, err
	}
	if err := sshSession.start(); err != nil {
		ErrorLog("NewSSHSession start error:%s", err.Error())
		return nil, err
	}
	sshSession.lastUseTime = time.Now()
	sshSession.brand = brand
	sshSession.manager = manager
	sshSession.sessionKey = config.GetSessionKey()
	return sshSession, nil
}

/**
 * 连接交换机，并打开session会话
 * @param user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口
 * @return 执行的错误
 * @author shenbowei
 */
func (this *SSHSession) createConnection(config *SSHConfig) error {
	DebugLog("Begin connect to %s", config.ipPort)
	client, err := ssh.Dial("tcp", config.ipPort, &ssh.ClientConfig{
		User: config.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 20 * time.Second,
		Config: ssh.Config{
			Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com",
				"arcfour256", "arcfour128", "aes128-cbc", "aes256-cbc", "3des-cbc", "des-cbc",
			},
		},
	})
	if err != nil {
		ErrorLog("SSH Dial to %s err:%s", config.ipPort, err.Error())
		return err
	}
	DebugLog("Connect to %s", config.ipPort)
	DebugLog("Begin new session %s", config.ipPort)
	session, err := client.NewSession()
	if err != nil {
		ErrorLog("NewSession err:%s", err.Error())
		return err
	}
	this.session = session
	DebugLog("New session created %s", config.ipPort)
	return nil
}

/**
 * 启动多线程分别将返回的两个管道中的数据传输到会话的输入输出管道中
 * @return 错误信息error
 * @author shenbowei
 */
 func (this *SSHSession) muxShell() error {
	defer func() {
		if err := recover(); err != nil {
			ErrorLog("SSHSession muxShell err:%s", err)
		}
	}()
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := this.session.RequestPty("vt100", 80, 40, modes); err != nil {
		ErrorLog("RequestPty error:%s", err)
		return err
	}
	w, err := this.session.StdinPipe()
	if err != nil {
		ErrorLog("StdinPipe() error:%s", err.Error())
		return err
	}
	r, err := this.session.StdoutPipe()
	if err != nil {
		ErrorLog("StdoutPipe() error:%s", err.Error())
		return err
	}

	in := make(chan string, 1024)
	out := make(chan string, 1024)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				ErrorLog("Goroutine muxShell write err:%s", err)
			}
		}()
		for cmd := range in {
			_, err := w.Write([]byte(cmd + "\n"))
			if err != nil {
				DebugLog("Writer write err:%s", err.Error())
				return
			}
		}
	}()

	go func() {
		defer func() {
			if err := recover(); err != nil {
				ErrorLog("Goroutine muxShell read err:%s", err)
			}
		}()
		var (
			buf [65 * 1024]byte
			t   int
		)
		for {
			n, err := r.Read(buf[t:])
			if err != nil {
				DebugLog("Reader read err:%s", err.Error())
				return
			}
			t += n
			out <- string(buf[:t])
			t = 0
		}
	}()
	this.in = in
	this.out = out
	return nil
}

/**
 * 开始打开远程ssh登录shell，之后便可以执行指令
 * @return 错误信息error
 * @author shenbowei
 */
func (this *SSHSession) start() error {
	if err := this.session.Shell(); err != nil {
		ErrorLog("Start shell error:%s", err.Error())
		return err
	}
	//等待登录信息输出
	this.ReadChannelExpect(time.Second, "#", ">", "]")
	return nil
}

func (sshSession *SSHSession) RunCmds(cmds ...string) (string, error) {
	sshSession.WriteChannel(cmds...)
	result := sshSession.ReadChannelTiming(2 * time.Second)
	return result, nil
}

func (sshSession *SSHSession) RunCmdsAndClose(cmds ...string) (string, error) {
	defer sshSession.Close()
	return sshSession.RunCmds(cmds...)
}

/**
 * 从输出管道中读取设备返回的执行结果，若输出流间隔超过timeout便会返回
 * @param timeout 从设备读取不到数据时的超时等待时间（超过超时等待时间即认为设备的响应内容已经被完全读取）
 * @return 从输出管道读出的返回结果
 * @author shenbowei
 */
 func (this *SSHSession) ReadChannelTiming(timeout time.Duration) string {
	DebugLog("ReadChannelTiming <wait timeout = %d>", timeout/time.Millisecond)
	output := ""
	isDelayed := 3

	for i := 0; i < 3000; i++ { //最多从设备读取300次，避免方法无法返回
		time.Sleep(time.Millisecond * 100) //每次睡眠0.1秒，使out管道中的数据能积累一段时间，避免过早触发default等待退出
		newData := this.readChannelData()
		DebugLog("ReadChannelTiming: read chanel buffer: %s", newData)
		if newData != "" {
			output += newData
			isDelayed = 10
			continue
		} else {
			DebugLog("ReadChannelTiming: delay for timeout: %d", isDelayed)
			isDelayed--
			time.Sleep(time.Second * 10)
			if (isDelayed == 0) {
				break
			}
		}
	}
	return output
}

/**
 * 从输出管道中读取设备返回的执行结果，若输出流间隔超过timeout或者包含expects中的字符便会返回
 * @param timeout 从设备读取不到数据时的超时等待时间（超过超时等待时间即认为设备的响应内容已经被完全读取）, expects...:期望得到的字符（可多个），得到便返回
 * @return 从输出管道读出的返回结果
 * @author shenbowei
 */
 func (this *SSHSession) ReadChannelExpect(timeout time.Duration, expects ...string) string {
	DebugLog("ReadChannelExpect <wait timeout = %d>", timeout/time.Millisecond)
	output := ""
	isDelayed := false
	for i := 0; i < 300; i++ { //最多从设备读取300次，避免方法无法返回
		time.Sleep(time.Millisecond * 100) //每次睡眠0.1秒，使out管道中的数据能积累一段时间，避免过早触发default等待退出
		newData := this.readChannelData()
		DebugLog("ReadChannelExpect: read chanel buffer: %s", newData)
		if newData != "" {
			output += newData
			isDelayed = false
			continue
		}
		for _, expect := range expects {
			if strings.Contains(output, expect) {
				return output
			}
		}
		//如果之前已经等待过一次，则直接退出，否则就等待一次超时再重新读取内容
		if !isDelayed {
			DebugLog("ReadChannelExpect: delay for timeout")
			time.Sleep(timeout)
			isDelayed = true
		} else {
			return output
		}
	}
	return output
}

/**
 * 清除管道缓存的内容，避免管道中上次未读取的残余内容影响下次的结果
 */
 func (this *SSHSession) readChannelData() string {
	output := ""
	for {
		time.Sleep(time.Millisecond * 100)
		select {
		case channelData, ok := <-this.out:
			if !ok {
				//如果out管道已经被关闭，则停止读取，否则<-this.out会进入无限循环
				return output
			}
			output += channelData
		default:
			return output
		}
	}
}


/**
 * SSHSession的关闭方法，会关闭session和输入输出管道
 * @author shenbowei
 */
 func (this *SSHSession) Close() {
	defer func() {
		if err := recover(); err != nil {
			ErrorLog("SSHSession Close err:%s", err)
		}
	}()
	if err := this.session.Close(); err != nil {
		ErrorLog("Close session err:%s", err.Error())
	}
	this.manager.DeleteSession(this.sessionKey)
	close(this.in)
	close(this.out)
}


/**
 * 获取最后的使用时间
 * @return time.Time
 * @author shenbowei
 */
 func (this *SSHSession) GetLastUseTime() time.Time {
	return this.lastUseTime
}

/**
 * 更新最后的使用时间
 * @author shenbowei
 */
func (this *SSHSession) UpdateLastUseTime() {
	this.lastUseTime = time.Now()
}


/**
 * 检查当前session是否可用
 * @return true:可用，false:不可用
 * @author shenbowei
 */
 func (this *SSHSession) CheckSelf() bool {
	defer func() {
		if err := recover(); err != nil {
			ErrorLog("SSHSession CheckSelf err:%s", err)
		}
	}()

	this.WriteChannel("\n")
	result := this.ReadChannelExpect(2*time.Second, "#", ">", "]")
	if strings.Contains(result, "#") ||
		strings.Contains(result, ">") ||
		strings.Contains(result, "]") {
		return true
	}
	return false
}

/**
 * 向管道写入执行指令
 * @param cmds... 执行的命令（可多条）
 * @author shenbowei
 */
 func (this *SSHSession) WriteChannel(cmds ...string) {
	DebugLog("WriteChannel <cmds=%v>", cmds)
	for _, cmd := range cmds {
		this.in <- cmd
	}
}