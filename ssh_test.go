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