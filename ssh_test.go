package switchgo

import "testing"

func TestLog(t *testing.T) {
	DebugLog("debug something")
	InfoLog("information for something")
	ErrorLog("Get some error. Ossss")
}