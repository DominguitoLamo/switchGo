package switchgo

import (
	"testing"
	"fmt"
)

func TestIpFormatValid(t *testing.T) {
	if err := ipFormatValid("hello"); err != nil {
		fmt.Println(err.Error())
	}

	if err := ipFormatValid("192.168.vv.100"); err != nil {
		fmt.Println(err.Error())
	}

	if err := ipFormatValid("192.168.300.100"); err != nil {
		fmt.Println(err.Error())
		return
	}
}

func TestLog(t *testing.T) {
	DebugLog("debug something")
	InfoLog("information for something")
	ErrorLog("Get some error. Ossss")
}

func TestSSHConfigCreate(t *testing.T) {
	sshConfig, err := SSHConfigCreate("gpmadmin", "aaa", "10.3.1.10", "22", "cisco")
	if (err != nil) {
		fmt.Println(err.Error())
	}
	fmt.Println(sshConfig)
}

func TestSessionManager(t *testing.T) {
	sessionManager := NewSessionManager()
	config, _ := SSHConfigCreate("gpmadmin", "Iw30#c61", "10.3.1.60", "22", "cisco")
	sessionManager.GetSSHSession(config)
}

func TestRunCmd(t *testing.T) {
	sessionManager := NewSessionManager()
	config, _ := SSHConfigCreate("gpmadmin", "Iw30#c61", "10.3.1.60", "22", "cisco")
	session, _ := sessionManager.GetSSHSession(config)
	result, _ := session.RunCmdsAndClose("en", "Iw30#c61", "sh running-config")
	fmt.Println(result)
}

func TestMultipleRunCmds(t *testing.T) {
	sessionManager := NewSessionManager()
	config1, _ := SSHConfigCreate("netcraft", "Iw30#c61", "10.3.1.60", "22", "cisco")
	config2, _ := SSHConfigCreate("netcraft", "Iw30#c61", "10.3.1.61", "22", "cisco")
	config3, _ := SSHConfigCreate("netcraft", "Iw30#c61", "10.3.1.62", "22", "cisco")
	configs := make([]*SSHConfig, 3)
	configs = append(configs, config1, config2, config3)
	c := make(chan string, 3)
	for _, config := range configs {
		go func() {
			session, _ := sessionManager.GetSSHSession(config)
			result, _ := session.RunCmdsAndClose("en", "Iw30#c61", "sh running-config")
			c <- result
		}()
	}

	fmt.Println(<-c)
}