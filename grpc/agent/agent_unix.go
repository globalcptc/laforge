// +build unix linux

package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	s "os"
	"os/exec"
	os "os/user"
	"syscall"
)

func MD5Sum(content string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(content)))
}

// RebootSystem Reboots Host Operating System
func RebootSystem() {
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}

// CreateSystemUser Create a new User
func CreateSystemUser(username string, password string) error {
	_, err := os.Lookup(username)
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

// Validation functions
func GetNetBanner(portnum int64) (bool, error) { // exists (boolean)
	return true, nil
}

func NetHttpContentRegex(full_url string) (string, error) {
	net_resp, err := http.Get(full_url)
	if err != nil {
		return "", err
	}
	defer net_resp.Body.Close()
	page_html, deserialize_err := ioutil.ReadAll(net_resp.Body)
	if deserialize_err != nil {
		return "", deserialize_err
	}

	return MD5Sum(fmt.Sprintf("%s", page_html)), nil
}

func FileExists(file_location string) (bool, error) { // exists (boolean)
	stat_info, read_err := s.Stat(file_location)
	if read_err != nil {
		return false, read_err
	}
	return !stat_info.IsDir(), nil
}

func FileHash(file_location string) (string, error) { // exists (boolean)
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

func main() {
	fmt.Println("wew")
	// fmt.Println(NetHttpContentRegex("https://vcu.edu"))
	// fmt.Println(FileExists("/home/piero/most-coding-stuff/laforge/validations"))
	fmt.Println(FileHash("/home/piero/most-coding-stuff/laforge/validations/test_file.txt"))
}
