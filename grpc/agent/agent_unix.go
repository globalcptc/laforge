//go:build unix || linux
// +build unix linux

package main

import (
	"context"
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
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func MD5Sum(content string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(content)))
}

// RebootSystem Reboots Host Operating System
func RebootSystem() {
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)

	time.Sleep(1 * time.Hour) // sleep forever bc we need to restart
}

// CreateSystemUser Create a new User
func CreateSystemUser(username string, password string) error {
	_, err := user.Lookup(username)
	if err != nil {
		// ExecuteCommand("useradd", username)
		ChangeSystemUserPassword(username, password)
	}
	return nil
}

// ChangeSystemUserPassword Change user password.
func ChangeSystemUserPassword(username string, password string) error {
	cmd := exec.Command("passwd", username)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		// logger.Error(err)
	}
	defer stdin.Close()

	passnew := fmt.Sprintf("%s\n%s\n", password, password)

	io.WriteString(stdin, passnew)

	if err = cmd.Start(); err != nil {
		// logger.Errorf("An error occured: ", err)
	}

	cmd.Wait()

	return nil
}

// AddSystemUserGroup Change user password.
func AddSystemUserGroup(groupname string, username string) error {
	// ExecuteCommand("usermod", "-a", "-G", groupname, username)
	return nil
}

// SystemDownloadFile Download a file with OS specific file endings
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

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

// SystemExecuteCommand Runs the Command that is inputted and either returns the error or output
func SystemExecuteCommand(command string, args ...string) (string, error) {
	var err error
	_, err = os.Stat(command)
	// output := ""
	if err == nil {
		// Make sure we have rwx permissions if it's a script
		err = os.Chmod(command, 0700)
		if err != nil {
			return "", err
		}
	}
	// Execute the command
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
	// retryCount := 5
	// for i := 0; i < retryCount; i++ {
	// 	// Get the data
	// 	cmd := exec.Command(command, args...)
	// 	stdout, err := cmd.StdoutPipe()
	// 	if err != nil {
	// 		fmt.Printf("error piping stdout: %v", err)
	// 		continue
	// 	}
	// 	stderr, err := cmd.StderrPipe()
	// 	if err != nil {
	// 		fmt.Printf("error piping stderr: %v", err)
	// 		continue
	// 	}
	// 	err = cmd.Run()
	// 	// out, err := cmd.CombinedOutput()
	// 	if err == nil {
	// 		// output = string(out)
	// 		combinedOutput := io.MultiReader(stdout, stderr)
	// 		var buff []byte
	// 		_, err = combinedOutput.Read(buff)
	// 		if err != nil {
	// 			fmt.Printf("error reading combined output: %v", err)
	// 			continue
	// 		}
	// 		output = string(buff)
	// 		break
	// 	}
	// 	time.Sleep(1 * time.Minute)
	// }
	// if err != nil {
	// 	return output, err
	// }
	// _, err = cmd.Output()
	// if err != nil {
	// 	return err
	// }
	// return string(output)
	// return output, nil
}

func GetSystemDependencies() []string {
	return []string{
		"Requires=network.target",
		"After=network-online.target"}
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

func FileExists(file_location string) (bool, error) { // exists (boolean)
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
	_, err := io.Copy(file_hash, file_read)
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%x", file_hash.Sum(nil)), nil
}

func DirectoryExists(directory string) (bool, error) { // exists (boolean)
	stat_info, read_err := s.Stat(directory)
	if read_err != nil {
		return false, read_err
	}
	return stat_info.IsDir(), nil
}

func UserExists(user_name string) (bool, error) { // exists (boolean)
	passwd_contents, read_err := ioutil.ReadFile("/etc/passwd")
	if read_err != nil {
		return false, read_err
	}
	passwd_text := strings.Split(string(passwd_contents), "\n")
	for i := 0; i < len(passwd_text); i++ {
		if strings.HasPrefix(passwd_text[i], user_name) {
			return true, nil
		}
	}
	return false, nil
}

func UserGroupMember(user_name string, group_name string) (bool, error) { // is in the group or not (boolean)
	group_contents, read_err := ioutil.ReadFile("/etc/group")
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
		group_line_chunks := strings.Split(groups[i], ":")
		if group_line_chunks[0] == group_name && len(group_line_chunks) > 3 {
			// first part of /etc/group entry matches and there are users assigned to the group
			users_in_group := strings.Split(group_line_chunks[3], ",")
			for j := 0; j < len(users_in_group); j++ {
				if users_in_group[j] == user_name {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// https://stackoverflow.com/questions/56336168/golang-check-tcp-port-open
func HostPortOpen(port int64) (bool, error) { // exists (boolean)

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port))), 10*time.Second)
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
	result := exec.Command("ps", "-a")
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

// Adapted from https://stackoverflow.com/questions/48263281/how-to-find-sshd-service-status-in-golang
func HostServiceState(service_name string) (string, error) {
	// returned status is one of the following:
	// active | inactive | enabled | disabled | static | masked | alias | linked
	// https://www.cyberciti.biz/faq/systemd-systemctl-view-status-of-a-service-on-linux/ lists all possibilities and meanings
	cmd := exec.Command("systemctl", "check", "sshd") // ASSUMPTION: the computer uses systemd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func LinuxAPTInstalled(package_name string) (bool, error) { // installed
	result := exec.Command("apt", "-qq", "list", package_name)
	ps_output, err := result.Output()
	if err != nil {
		return false, err
	}
	apt_lines := strings.Split(string(ps_output), "\n")
	for i := 0; i < len(apt_lines); i++ {
		if strings.HasPrefix(apt_lines[i], package_name) && (strings.HasSuffix(apt_lines[i], "[installed]") || strings.HasSuffix(apt_lines[i], "[installed,local]") || strings.HasSuffix(apt_lines[i], "[installed,automatic]")) {
			return true, nil
		}
	}
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
	recv_chan := make(chan bool)
	go UDPOpenTest(conn, recv_chan, open_socket_payload)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	select {
	case <-recv_chan:
		return true, nil
	case <-time.After(10 * time.Second):
		return false, nil
	case <-ctx.Done():
		return false, nil
	}
}

func UDPOpenTest(socket net.Conn, return_chan chan bool, optional_payload string) {
	socket.Write([]byte(optional_payload))
	socket.Read([]byte(""))
	return_chan <- true
}

func NetICMP(ip string) (bool, error) { // responded (boolean)
	// This WILL block the thread! However, agent tasks are on their own threads.
	result := exec.Command("ping", "-c", "5", ip) // you can write a ping packet and send it using pure golang, however it's quite complicated and requires more importing of libraries
	ps_output, err := result.Output()
	if err != nil {
		return false, err
	}
	fmt.Println(string(ps_output))
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

func main() {
	// fmt.Println(NetHttpContentRegex("https://vcu.edu"))
	// fmt.Println(FileExists("/home/piero/most-coding-stuff/laforge/test_file")) // change to dir, won't get tripped up
	// fmt.Println(FileHash("/home/piero/most-coding-stuff/laforge/test_file.txt"))
	// fmt.Println(UserGroupMember("piero", "wew"))
	// fmt.Println(HostPortOpen(8080))
	// fmt.Println(HostProcessRunning("nginx"))
	// fmt.Println(HostServiceState("nginx"))
	// fmt.Println(LinuxAPTInstalled("wget"))
	// fmt.Println(NetTCPOpen("127.0.0.1", 80)) // test with nginx for ease
	// fmt.Println(NetICMP("192.168.1.255"))
	// fmt.Println(FileContentString("/home/piero/most-coding-stuff/laforge/test_file.txt", "hi"))
	// fmt.Println(FilePermission("/home/piero/most-coding-stuff/laforge/test_file.txt"))
	fmt.Println(NetUDPOpen("127.0.0.1", 3000, ""))
}
