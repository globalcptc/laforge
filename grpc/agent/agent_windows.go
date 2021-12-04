//go:build windows
// +build windows

package main

import (
	"fmt"
	s "os"
	"syscall"

	wapi "github.com/iamacarpet/go-win64api"
)

// RebootSystem Reboots Host Operating System
func RebootSystem() {
	// This is how to properlly rebot windows
	user32 := syscall.MustLoadDLL("user32")
	defer user32.Release()

	exitwin := user32.MustFindProc("ExitWindowsEx")

	r1, _, _ := exitwin.Call(0x02, 0)
	if r1 != 1 {
		ExecuteCommand("cmd", "/C", "shutdown", "/r")
	}
}

// CreateSystemUser Creates User with specified password.
func CreateSystemUser(username string, password string) error {
	_, err := wapi.UserAdd(username, username, password)
	return err
}

// ChangeSystemUserPassword Change user password.
func ChangeSystemUserPassword(username string, password string) error {
	_, err := wapi.ChangePassword(username, password)
	return err
}

// AddSystemUserGroup Add user to group.
func AddSystemUserGroup(groupname string, username string) error {
	_, err := wapi.LocalGroupAddMembers(groupname, []string{username})
	return err
}

// Validate file system

// Validation functions
func GetNetBanner(portnum int64) (bool, error) { // exists (boolean)
	return true, nil
}

func NetHttpContentRegex(full_url string) (string, error) {

	return "false", nil
}

func FileExists(file_location string) (bool, error) {
	stat_info, read_err := s.Stat(file_location)
	if read_err != nil {
		return false, read_err
	}
	return !stat_info.IsDir(), nil
}

func main() {
	fmt.Println("windows")
	fmt.Println(FileExists("C:\\Users\\The Power\\Documents\\2021Fall\\CMSC451\\LaForge\\laforge\\grpc\\agent\\agent_windows.go"))
}
