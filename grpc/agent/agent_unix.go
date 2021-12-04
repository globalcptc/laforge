// +build unix linux

package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	s "os"
	"os/exec"
	"os/user"
	"path/filepath"
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

func main() { // driver for tests, main function not necessary later
	fmt.Println("wew")
	// fmt.Println(NetHttpContentRegex("https://vcu.edu"))
	// fmt.Println(FileExists("/home/piero/most-coding-stuff/laforge/validations"))
	fmt.Println(FileHash("/home/piero/most-coding-stuff/laforge/validations/test_file.txt"))
}
