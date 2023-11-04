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
	sshConfig, err := SSHConfigCreate("gpmadmin", "aaa", "10.3.1.10", "22")
	if (err != nil) {
		fmt.Println(err.Error())
	}
	fmt.Println(sshConfig)
}

func TestNewSSHSession(t *testing.T) {
	sshConfig, err := SSHConfigCreate("gpmadmin", "Iw30#c61", "10.3.1.60", "22")
	if (err != nil) {
		fmt.Println(err.Error())
	}
	session, err := NewSSHSession(sshConfig, CISCO)
	fmt.Println(session.brand)
}

func TestSessionManager(t *testing.T) {
	sessionManager := NewSessionManager()
	config, _ := SSHConfigCreate("gpmadmin", "Iw30#c61", "10.3.1.60", "22")
	sessionManager.GetSSHSession(config, CISCO)
}