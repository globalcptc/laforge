//go:build windows
// +build windows

package agent

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	s "os"
	"os/exec"
	user "os/user"
	"regexp"

	"path/filepath"
	"strconv"
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
		SystemExecuteCommand("cmd", "/C", "shutdown", "/r", "/f")
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

func SystemDownloadFile(path, url, is_txt string) error {
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

	if is_txt == "true" {
		// Convert Unix line endings to windows line endings
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		body := strings.Replace(string(bodyBytes), "\n", "\r\n", -1)

		// Write the body to file
		_, err = io.WriteString(out, body)
	} else {
		// Write the body to file
		_, err = io.Copy(out, resp.Body)
	}

	return err
}

// SystemExecuteAnsible Runs Ansible Playbook
func SystemExecuteAnsible(playbookPath, connectionMethod, inventoryList string) (string, error) {
	return "", fmt.Errorf("Not Implemented for Windows")
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
}

func GetSystemDependencies() []string {
	return []string{}
}

// Validation functions
func GetNetBanner(portnum int64) (bool, error) { // exists (boolean)
	return true, nil
}

func NetHttpContentRegex(full_url string) (string, error) { // content hash (string)
	net_resp, err := http.Get(full_url)
	if err != nil {
		return "", err
	}
	defer net_resp.Body.Close()
	page_html, deserialize_err := ioutil.ReadAll(net_resp.Body)
	if deserialize_err != nil {
		return "", deserialize_err
	}

	// return MD5Sum(fmt.Sprintf("%s", page_html)), nil
	return string(page_html[:]), nil // stringify
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

func FileContentRegex(file_location string, pattern string) (bool, error) { // page content to be returned and checked serverside (string)
	content, err := ioutil.ReadFile(file_location)
	if err != nil {
		return false, err
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	return regex.Match(content), nil
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
		if user.Username == user_name {
			return true, nil
		}
	}
	return false, nil
}

func HostPortOpen(port int) (bool, error) {
	result := exec.Command("netstat", "-na")
	ps_output, err := result.Output()
	if err != nil {
		return false, err
	}
	ps_lines := strings.Split(string(ps_output), "\n")
	for i := 0; i < len(ps_lines); i++ {
		if strings.Contains(ps_lines[i], " localhost:"+strconv.Itoa(port)+" ") || strings.Contains(ps_lines[i], " 0.0.0.0:"+strconv.Itoa(port)+" ") || strings.Contains(ps_lines[i], " 127.0.0.1:"+strconv.Itoa(port)+" ") {
			return true, nil
		}
	}
	return false, nil
}

func HostProcessRunning(process_name string) (bool, error) { // running (boolean)
	result := exec.Command("tasklist")
	tasklist_output, err := result.Output()
	if err != nil {
		return false, err
	}
	tasklist_lines := strings.Split(string(tasklist_output), "\n")
	for i := 0; i < len(tasklist_lines); i++ {
		if strings.Contains(tasklist_lines[i], process_name) {
			return true, nil
		}
	}
	return false, nil
}

func HostServiceState(service_name string, service_status string) (bool, error) {
	return false, nil
}

func NetTCPOpen(ip string, port int) (bool, error) { // exists (boolean)
	// net.Dial or net.DialTimeout will succeed if the following succeeds:
	/*
	   Client -> Server: SYN
	   Server -> Client: SYN-ACK
	   Client -> Server: ACK
	*/
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), 10*time.Second)
	if err != nil && !strings.HasSuffix(err.Error(), "connection refused") {
		return false, err
	}
	if conn != nil {
		defer conn.Close() // no hanging processes
		return true, nil
	} else {
		return false, nil
	}
}

func NetUDPOpen(ip string, port int, open_socket_payload string) (bool, error) { // exists (boolean)
	conn, err := net.DialTimeout("udp", net.JoinHostPort(ip, strconv.Itoa(port)), 10*time.Second)
	// we don't really know if a udp connection is alive or not, so
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

func NetICMP(ip string) (bool, error) { // responded (boolean)
	// This WILL block the thread! However, agent tasks are on their own threads.
	result := exec.Command("ping", "-c", "5", ip) // you can write a ping packet and send it using pure golang, however it's quite complicated and requires more importing of libraries
	ps_output, err := result.Output()
	if err != nil {
		return false, err
	}
	ps_lines := strings.Split(string(ps_output), "\n")
	for i := 0; i < len(ps_lines); i++ {
		if strings.HasPrefix(ps_lines[i], "5 packets transmitted, 5 received") { // this is pretty jank
			return true, nil
		}
	}
	return false, nil
}

func FileContentString(filepath string, text string) (bool, error) { // matches
	file_contents, read_err := ioutil.ReadFile(filepath)
	if read_err != nil {
		return false, read_err
	}
	lines := strings.Split(string(file_contents), "\n")
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], text) {
			return true, nil
		}
	}
	return false, nil
}

// https://stackoverflow.com/questions/45429210/how-do-i-check-a-files-permissions-in-linux-using-go
func FilePermission(filepath string) (string, error) { // permissions (in the form of SRWXRWXRWX, where S is setuid bit)
	info, err := os.Stat(filepath)
	if err != nil {
		return "", err
	}
	return info.Mode().String(), nil
}

// https://go.dev/src/os/user/lookup_windows.go
// https://cs.opensource.google/go/go/+/refs/tags/go1.17.7:src/os/user/lookup.go
func UserGroupMember(user_name string, group_name string) (bool, error) { // is in the group or not (boolean)
	// fmt.Println(os.Hostname())

	//returns Uid, Gid, Username, Name and HomeDir
	u, err := user.Lookup(user_name)
	if err != nil {
		return false, err
	}

	// returns string array of Gids of specific user
	userGroups, err := u.GroupIds()
	if err != nil {
		return false, nil
	}

	//returns Gid and Name
	group, err := user.LookupGroup(group_name)
	if err != nil {
		return false, nil
	}

	for i := range userGroups {
		if group.Gid == userGroups[i] {
			// Found a Gid that matches Gid of group_name
			return true, nil
		}
	}

	return false, nil
}

func NetHttpContentHash(full_url string, hashed_url string) (bool, error) { // content hash (string)
	net_resp, err := http.Get(full_url)
	if err != nil {
		return false, err
	}
	defer net_resp.Body.Close()
	page_html, deserialize_err := ioutil.ReadAll(net_resp.Body)
	if deserialize_err != nil {
		return false, deserialize_err
	}

	hashed := md5.Sum([]byte(page_html))
	hashed_string := fmt.Sprintf("%x", hashed)

	result := strings.EqualFold(hashed_string, hashed_url)
	if result {
		return true, nil
	}

	return false, nil

	// return string(page_html[:]), nil // stringify
}

func HostFirewallPort(port int) (bool, error) {
	result := exec.Command("netsh", "firewall", "show", "state")
	ps_output, err := result.Output()
	if err != nil {
		return false, err
	}
	ps_lines := strings.Split(string(ps_output), "\n")
	for i := 0; i < len(ps_lines); i++ {
		if strings.Contains(ps_lines[i], strconv.Itoa(port)+" ") {
			return true, nil
		}
	}
	return false, nil
}

func LinuxAPTInstalled(package_name string) (bool, error) {
	return false, nil
}
