//go:build windows
// +build windows

// package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	s "os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
		ExecuteCommand("cmd", "/C", "shutdown", "/r", "/f")
	}

	time.Sleep(1 * time.Hour) // sleep forever bc we need to restart
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

func SystemDownloadFile(path, url string) error {
	retryCount := 5
	var resp *http.Response
	var err error
	for i := 0; i < retryCount; i++ {
		// Get the data
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Create the file
	out, err := os.Create(absolutePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Convert Unix line endings to windows line endings
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := strings.Replace(string(bodyBytes), "\n", "\r\n", -1)

	// Write the body to file
	_, err = io.WriteString(out, body)
	return err
}

// SystemExecuteCommand Runs the Command that is inputted and either returns the error or output
func SystemExecuteCommand(command string, args ...string) (string, error) {
	var err error
	_, err = os.Stat(command)
	output := ""
	if err == nil {
		// Make sure we have rwx permissions if it's a script
		err = os.Chmod(command, 0700)
		if err != nil {
			return output, err
		}
	}
	// Execute the command
	arguments := []string{}
	arguments = append(arguments, command)
	arguments = append(arguments, args...)
	cmd := exec.Command("powershell.exe", arguments...)
	out, err := cmd.CombinedOutput()
	return string(out), err
	// retryCount := 5
	// for i := 0; i < retryCount; i++ {
	// 	// Get the data
	// 	cmd := exec.Command("powershell.exe", arguments...)
	// 	out, err := cmd.CombinedOutput()
	// 	if err == nil {
	// 		output = string(out)
	// 		break
	// 	}
	// 	time.Sleep(1 * time.Minute)
	// }
	// if err != nil {
	// 	return output, err
	// }
	// return output, nil
}

func GetSystemDependencies() []string {
	return []string{}
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

func FileHash(file_location string) (string, error) { // hash of the file (string)
	file_read, read_err := s.Open(file_location)
	if read_err != nil {
		return "", read_err
	}
	file_hash := md5.New()
	if _, err := io.Copy(file_hash, file_read); err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%x", file_hash.Sum(nil)), nil
}

func FileContentRegex(file_location string) (string, error) { // page content to be returned and checked serverside (string)
	file_read, read_err := s.Open(file_location)
	if read_err != nil {
		return "", read_err
	}
	file_hash := md5.New()
	if body_contents, err := io.Copy(file_hash, file_read); err != nil {
		log.Fatal(err)
	}
	return body_contents, nil
}

func DirectoryExists(directory string) (bool, error) { // exists (boolean)
	stat_info, read_err := s.Stat(directory)
	if read_err != nil {
		return false, read_err
	}
	return stat_info.IsDir(), nil
}

func UserExists(user_name string) (bool, error) { // exists (boolean
	users, err := wapi.ListLocalUsers()
	if err != nil {
		return false, err
	}
	for _, user := range users {
		if u.Username == user_name {
			return true, nil
		}
	}
	return false, nil
}

func UserGroupMember(user_name string, group_name string) (bool, error) { // is in the group or not (boolean)
	// TODO: need a special way to get this via windows
	// group_contents, read_err := ioutil.ReadFile("/etc/group")
	if read_err != nil {
		return false, read_err
	}
	groups := strings.Split(string(group_contents), "\n")
	for i := 0; i < len(groups); i++ {
		// example groups
		/*
			adm:x:4:piero
			tty:x:5:
			disk:x:6:
			lp:x:7:
			mail:x:8:
			news:x:9:
			uucp:x:10:
			man:x:12:
			proxy:x:13:
			kmem:x:15:
			dialout:x:20:
			fax:x:21:
			voice:x:22:
			cdrom:x:24:piero
		*/
		group_line_chunks := strings.Split(string(group_contents), ":")
		if group_line_chunks[0] == group_name && len(group_line_chunks) > 3 {
			// first part of /etc/group entry matches and there are users assigned to the group
			users_in_group := strings.Split(group_line_chunks[3], ",")
			for j := 0; j < len(users_in_group); j++ {
				if users_in_group[j] == user_name {
					return true, nil
				}
			}
		}
		return false, nil
	}
	return false, nil
}

// https://stackoverflow.com/questions/56336168/golang-check-tcp-port-open
func HostPortOpen(port int64) (bool, error) { // exists (boolean)
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", string(port)), time.Second)
	if err != nil {
		return false, err
	}
	if conn != nil {
		defer conn.Close() // no hanging processes
		return true, nil
	} else {
		return false, nil
	}
}

func HostProcessRunning(process_name string) (bool, error) { // running (boolean)
	// TODO: add specific logic for windows
	// result := exec.Command("sh", "ps", "-a")
	ps_output, err := result.Output()
	if err != nil {
		return false, err
	}
	ps_lines := strings.Split(string(ps_output), "\n")
	for i := 0; i < len(ps_lines); i++ {
		if strings.HasSuffix(ps_lines[i], process_name) {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	fmt.Println("windows")
	fmt.Println(FileExists("C:\\Users\\The Power\\Documents\\2021Fall\\CMSC451\\LaForge\\laforge\\grpc\\agent\\agent_windows.go"))
}
