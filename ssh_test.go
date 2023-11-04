package switchgo

import (
	"testing"
	"fmt"
)

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