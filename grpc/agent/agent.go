package main

//go:generate fileb0x assets.toml
import (
	"context"
	"crypto/md5"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gen0cide/laforge/grpc/agent/static"
	pb "github.com/gen0cide/laforge/grpc/proto"
	"github.com/kardianos/service"
	"github.com/mholt/archiver"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	TaskFailed    = "FAILED"
	TaskRunning   = "INPROGRESS"
	TaskSucceeded = "COMPLETE"
	LogError 	  = "ERROR"
	LogWarning    = "WARNING"
	LogInfo       = "INFO"
)


var (
	logger  service.Logger
	address = "localhost:50051"
	// defaultName      = "Laforge Agent"
	certFile         = "service.pem"
	heartbeatSeconds = 10
	clientID         = "1"
	previousTask     = ""
)

// Program structures.
//  Define Start and Stop methods.
type program struct {
	exit chan struct{}
}

// Start What is Run when the executable is started up.
func (p *program) Start(s service.Service) error {
	p.exit = make(chan struct{})

	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

// ExecuteCommand Runs the Command that is inputted and either returns the error or output
func ExecuteCommand(command string, args ...string) (string, error) {
	return SystemExecuteCommand(command, args...)
}

// DeleteObject Deletes the Object that is inputted and either returns the error or nothing
func DeleteObject(file string) error {
	err := os.RemoveAll(file)
	if err != nil {
		return err
	}
	return nil
}

// Reboot Reboots Host Operating System
func Reboot() {
	RebootSystem()
}

// ExtractArchive will extract archive to foler path.
func ExtractArchive(filepath string, folderpath string) error {
	err := archiver.Unarchive(filepath, folderpath)
	return err
}

// CreateUser will create a new user.
func CreateUser(username string, password string) error {
	return CreateSystemUser(username, password)
}

// ChangeUserPassword will change the users password
func ChangeUserPassword(username string, password string) error {
	return ChangeSystemUserPassword(username, password)
}

// AddUserGroup will extract archive to foler path.
func AddUserGroup(groupname string, username string) error {
	return AddSystemUserGroup(groupname, username)
}

// DownloadFile will download a url to a local file.
func DownloadFile(path, url, is_txt string) error {
	return SystemDownloadFile(path, url, is_txt)
}

// ExecuteAnsible will execute an Ansible Playbook
func ExecuteAnsible(playbookPath, connectionMethod, inventoryList string) (string, error) {
	return SystemExecuteAnsible(playbookPath, connectionMethod, inventoryList)
}

// ChangePermissions will download a url to a local file.
func ChangePermissions(path string, perms int) error {
	var err error
	_, err = os.Stat(path)
	if err == nil {
		// Make sure we have rwx permissions if it's a script
		err = os.Chmod(path, os.FileMode(perms))
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

// AppendFile will download a url to a local file.
func AppendFile(path string, content string) error {
	var err error
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return err
	}
	return nil
}

// ValidateMD5Hash Validates the MD5 Hash of a file with the provided MD5 Hash
func ValidateMD5Hash(filepath string, md5hash string) error {
	var calculatedMD5Hash string

	// Open the file
	file, err := os.Open(filepath)

	// Can't open the file, assuming false
	if err != nil {
		return err
	}

	// Close the file when we're done
	defer file.Close()

	// Open a new hash interface
	hash := md5.New()

	// Hash the file
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	byteHash := hash.Sum(nil)[:16]

	// Convert bytes to string
	calculatedMD5Hash = hex.EncodeToString(byteHash)

	if calculatedMD5Hash == md5hash {
		return errors.New("MD5 hashes do not match")
	} else {
		return nil
	}
}

// RequestTask Function Requests task from the GRPC server to be run on the client
func RequestTask(c pb.LaforgeClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request := &pb.TaskRequest{ClientId: clientID}
	r, err := c.GetTask(ctx, request)

	if r.GetCommand() == pb.TaskReply_DEFAULT {
		logf(logger, LogError, "Received Empty Task %v", err)
		return
	} else if r.Id == previousTask {
		logf(logger, LogError, "Received Duplicate Task %s %v", r.Id, err)
		return
	}

	taskRequest := &pb.TaskStatusRequest{
		TaskId: r.GetId(),
		Status: TaskRunning,
	}
	c.InformTaskStatus(ctx, taskRequest)

	if err != nil {
		logf(logger, LogError, "Error Occured in Task Receipt: %v", err)
	} else {
		switch r.GetCommand() {
		case pb.TaskReply_ANSIBLE:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			playbookPath := taskArgs[0]
			connectionMethod := taskArgs[1]
			inventoryList := taskArgs[2]
			logf(logger, LogInfo, "Recevied Task (Ansible) - playbook: %s, method: %s, inv: %s", playbookPath, connectionMethod, inventoryList)
			taskoutput, taskerr := ExecuteAnsible(playbookPath, connectionMethod, inventoryList)
			taskoutput = strings.ReplaceAll(taskoutput, "\n", "🔥")
			RequestTaskStatusRequest(taskoutput, taskerr, r.Id, c)
		case pb.TaskReply_EXECUTE:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			command := taskArgs[0]
			args := taskArgs[1:]
			logf(logger, LogInfo, "Recevied Task (Execute) - Cmd: %s args: %v", command, args)
			taskoutput, taskerr := ExecuteCommand(command, args...)
			taskoutput = strings.ReplaceAll(taskoutput, "\n", "🔥")
			RequestTaskStatusRequest(taskoutput, taskerr, r.Id, c)
		case pb.TaskReply_DOWNLOAD:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			filepath := taskArgs[0]
			url := taskArgs[1]
			is_txt := taskArgs[2]
			logf(logger, LogInfo, "Recevied Task (Download) - From: %s To: %s", url, filepath)
			taskerr := DownloadFile(filepath, url, is_txt)
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		case pb.TaskReply_EXTRACT:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			filepath := taskArgs[0]
			folder := taskArgs[1]
			logf(logger, LogInfo, "Recevied Task (Extract) - Path: %s To: %s", filepath, folder)
			taskerr := ExtractArchive(filepath, folder)
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		case pb.TaskReply_DELETE:
			args := r.GetArgs()
			logf(logger, LogInfo, "Recevied Task (Delete) - Path: %s", args)
			taskerr := DeleteObject(args)
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		case pb.TaskReply_REBOOT:
			logf(logger, LogInfo, "Recevied Task (Reboot)")
			// taskRequest := &pb.TaskStatusRequest{TaskId: r.Id, Status: TaskSucceeded}
			// c.InformTaskStatus(ctx, taskRequest)
			// Reboot after telling server task succeeded
			RequestTaskStatusRequest("", nil, r.Id, c)
			Reboot()
		case pb.TaskReply_CREATEUSER:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			username := taskArgs[0]
			password := taskArgs[1]
			logf(logger, LogInfo, "Recevied Task (Create User) - user: %s, pass: %s", username, password)
			taskerr := CreateUser(username, password)
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		case pb.TaskReply_ADDTOGROUP:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			group := taskArgs[0]
			username := taskArgs[1]
			logf(logger, LogInfo, "Recevied Task (Add Group) - user: %s, group: %s", username, group)
			taskerr := AddUserGroup(group, username)
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		case pb.TaskReply_CREATEUSERPASS:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			username := taskArgs[0]
			password := taskArgs[1]
			logf(logger, LogInfo, "Recevied Task (Set Password) - user: %s, pass: %s", username, password)
			taskerr := ChangeUserPassword(username, password)
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		case pb.TaskReply_VALIDATE:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			filepath := taskArgs[0]
			md5hash := taskArgs[1]
			logf(logger, LogInfo, "Recevied Task (Validate Hash) - file: %s, hash: %s", filepath, md5hash)
			taskerr := ValidateMD5Hash(filepath, md5hash)
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		case pb.TaskReply_CHANGEPERMS:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			path := taskArgs[0]
			permsString := taskArgs[1]
			logf(logger, LogInfo, "Recevied Task (Change File Perms) - path: %s, perms: %s", path, permsString)
			perms, taskerr := strconv.Atoi(permsString)
			if taskerr == nil {
				taskerr = ChangePermissions(path, perms)
			}
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		case pb.TaskReply_APPENDFILE:
			taskArgs := strings.Split(r.GetArgs(), "💔")
			path := taskArgs[0]
			logf(logger, LogInfo, "Recevied Task (Append To File) - path: %s", path)
			content := strings.ReplaceAll(taskArgs[1], "🔥", "\n")
			taskerr := AppendFile(path, content)
			RequestTaskStatusRequest("", taskerr, r.Id, c)
		default:
			logf(logger, LogWarning, "Default Task Received: %v", r)
			RequestTaskStatusRequest("", nil, r.Id, c)
		}

		previousTask = r.Id
	}
}

// RequestTaskStatusRequest Tell the server the status of a completed task
func RequestTaskStatusRequest(taskoutput string, taskerr error, taskID string, c pb.LaforgeClient) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if taskerr != nil {
		logf(logger, LogError, "Error occured trying to request task status: %v", taskerr)
		taskRequest := &pb.TaskStatusRequest{TaskId: taskID, Status: TaskFailed, ErrorMessage: taskerr.Error(), Output: taskoutput}
		c.InformTaskStatus(ctx, taskRequest)
	} else {
		taskRequest := &pb.TaskStatusRequest{TaskId: taskID, Status: TaskSucceeded, ErrorMessage: "", Output: taskoutput}
		c.InformTaskStatus(ctx, taskRequest)
	}
}

// SendHeartBeat Send the GRPC server a Heartbeat with specified parameters
func SendHeartBeat(c pb.LaforgeClient, taskChannel chan *pb.HeartbeatReply) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := &pb.HeartbeatRequest{ClientId: clientID}
	hostInfo, hostErr := host.Info()
	if hostErr == nil {
		(*request).Hostname = hostInfo.Hostname
		(*request).Uptime = hostInfo.Uptime
		(*request).Boottime = hostInfo.BootTime
		(*request).Numprocs = hostInfo.Procs
		(*request).Os = hostInfo.OS
		(*request).Hostid = hostInfo.HostID
	}
	mem, memErr := mem.VirtualMemory()
	if memErr == nil {
		(*request).Totalmem = mem.Total
		(*request).Freemem = mem.Free
		(*request).Usedmem = mem.Used
	}
	load, loadErr := load.Avg()
	if loadErr == nil {
		(*request).Load1 = load.Load1
		(*request).Load5 = load.Load5
		(*request).Load15 = load.Load15
	}
	(*request).Timestamp = timestamppb.Now()
	r, err := c.GetHeartBeat(ctx, request)
	if err != nil {
		logf(logger, LogError, "Error occured trying to send heartbeat: %v", err)
	} else {
		taskChannel <- r
		logf(logger, LogInfo, "Got a heartbeat from server, task queue length: %d", len(taskChannel))
	}

}

// StartTaskRunner Gets a Heartbeat reply from the task channel, and if there are avalible tasks it will request them
func StartTaskRunner(c pb.LaforgeClient, taskChannel chan *pb.HeartbeatReply, doneChannel chan bool) {
	r := <-taskChannel
	logf(logger, LogInfo, "Starting Task Execution.  Remaining tasks in channel: %d.  Available Tasks: %s, Response: %s", len(taskChannel), r.GetStatus(), r.GetAvalibleTasks())

	if r.GetAvalibleTasks() {
		RequestTask(c)
	}

	doneChannel <- true
}

// genSendHeartBeat A goroutine that is called, which periodically send a heartbeat to the GRPC Server
func genSendHeartBeat(p *program, c pb.LaforgeClient, taskChannel chan *pb.HeartbeatReply) chan bool {
	// func genSendHeartBeat(p *program, c pb.LaforgeClient, taskChannel chan *pb.HeartbeatReply, wg *sync.WaitGroup) chan bool {
	ticker := time.NewTicker(time.Duration(heartbeatSeconds) * time.Second)
	stop := make(chan bool, 1)

	go func() {
		defer log(logger, LogInfo, "Heartbeat Ticker Function Stopped")
		for {
			select {
			case <-ticker.C:
				go SendHeartBeat(c, taskChannel)
			case <-p.exit:
				stop <- true
				return
			}
		}
	}()

	return stop
	// defer wg.Done()
	// for {
	// 	select {
	// 	case <-ticker.C:
	// 		SendHeartBeat(c, taskChannel)
	// 	case <-p.exit:
	// 		ticker.Stop()
	// 	}
	// }
}

// genStartTaskRunner A goroutine that is called, which checks responses from GRPC server for avalible tasks
func genStartTaskRunner(p *program, c pb.LaforgeClient, taskChannel chan *pb.HeartbeatReply) chan bool {
	// func genStartTaskRunner(p *program, c pb.LaforgeClient, taskChannel chan *pb.HeartbeatReply, wg *sync.WaitGroup) {
	ticker := time.NewTicker(time.Duration(heartbeatSeconds) * time.Second)
	stop := make(chan bool, 1)

	go func() {
		defer log(logger, LogInfo, "Task Runner function stopped")
		taskIsDone := make(chan bool, 1)
		// Kick off first task grab
		taskIsDone <- true
		for {
			select {
			case <-ticker.C:
				select {
				case <-taskIsDone:
					go StartTaskRunner(c, taskChannel, taskIsDone)
				default:
					log (logger, LogInfo, "Task current in progress, waiting until completion.")
				}
			case <-p.exit:
				stop <- true
				return
			}
		}
	}()

	return stop
}

// run Function that is called when the program starts and run all the Go Routines
func (p *program) run() error {
	logf(logger, LogInfo, "LaForge Service Starting.  Platform: %v", service.Platform())
	// var wg sync.WaitGroup

	// TLS Cert for verifying GRPC Server
	certPem, certerr := static.ReadFile(certFile)
	if certerr != nil {
		fmt.Println("File reading error", certerr)
		return nil
	}

	// Starts GRPC Connection with cert included in the binary
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certPem)
	creds := credentials.NewClientTLSFromCert(certPool, "")
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))

	if err != nil {
		logf(logger, LogError, "Could not connect to LaForge Server: %v", err)
	}
	defer conn.Close()
	c := pb.NewLaforgeClient(conn)

	// START VARS
	taskChannel := make(chan *pb.HeartbeatReply)
	// wg.Add(2)
	heartbeatDone := genSendHeartBeat(p, c, taskChannel)
	taskRunnerDone := genStartTaskRunner(p, c, taskChannel)

	<-heartbeatDone
	<-taskRunnerDone
	// wg.Wait()
	return nil
}

// Stop Called when the Agent is closed
func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	log(logger, LogWarning, "LaForge Service Stopping")
	close(p.exit)
	return nil
}

// Service setup.
//   Define service config.
//   Create the service.
//   Setup the logger.
//   Handle service controls (optional).
//   Run the service.
func main() {
	svcFlag := flag.String("service", "", "Control the system service.")
	flag.Parse()

	options := make(service.KeyValue)
	options["Restart"] = "always"
	options["OnFailure"] = "restart"
	// options["SuccessExitStatus"] = "1 2 8 SIGKILL"
	svcConfig := &service.Config{
		Name:         "laforge-agent",
		DisplayName:  "Laforge Agent",
		Description:  "Tool used for monitoring hosts. NOT IN COMPETITION SCOPE",
		Dependencies: GetSystemDependencies(),
		Option:       options,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Error(err)
	}
	errs := make(chan error, 5)
	logger, err = s.Logger(errs)
	if err != nil {
		logger.Error(err)
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				logger.Error(err)
			}
		}
	}()

	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			logf(logger, LogError, "Service used invalid actions: %q %v", service.ControlAction, err)
		}
		return
	}
	err = s.Run()
	if err != nil {
		logf(logger, LogError, "Service unable to successfully start: %v", err)
	}
}
