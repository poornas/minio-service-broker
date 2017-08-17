package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/minio/minio-service-broker/utils"

	"code.cloudfoundry.org/lager"
)

const (
	// Root directory where agent runs
	RootDir = "/tmp/data/"
	// Config directory where app resides
	ConfigDir = "/tmp/data/{app name}/config"
	// Data directory where buckets are created
	DataDir = "/tmp/data/{app name}/data"
	// Hard code ip address of server running service agent for now
)

type ServiceState struct {
	port   int
	status string // Server on|off
	pid    int    // process id of running service
}

// MinioServiceAgent holds the map of service name to status TODO => Persist agent config to some config.json
type MinioServiceAgent struct {
	log      lager.Logger
	conf     utils.Config
	services map[string]*ServiceState
	rootURL  string
}

type ServerConfig struct {
}

// get config from json file and hydrate it
func getConfig(path string) ServerConfig {
	return ServerConfig{}
}

// Return an available free port
func getFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

//CreateInstanceHandler creates an instance of minio server
func (agent *MinioServiceAgent) CreateInstanceHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	instanceID := vars["instance-id"]
	log.Info("create instance!!!!!!::" + r.RequestURI + "::" + instanceID)
	// Spawn minio instance
	fmt.Println("about to spawn minio server....")
	_, err := exec.LookPath("minio")
	if err != nil {
		agent.log.Info("minio binary not found in install paths")
	}
	port := getFreePort()

	// minio directory path
	dirPath := RootDir + instanceID + "/" + "data"
	configDirPath := RootDir + instanceID + "/" + "config"
	cmd := exec.Command("minio", "server", "--address", ":"+strconv.Itoa(port), "--config-dir", configDirPath, dirPath)

	f, err := os.Open("/tmp/log")
	if err != nil {
		agent.log.Fatal("Failed to provision an instance", err)
	}
	cmd.Stdout = f
	cmd.Stderr = f
	// Execute command

	err = cmd.Start() // will wait for command to return

	// Only output the commands stdout

	if err != nil {
		agent.log.Fatal("Failed to provision instance", err)
	}
	fmt.Println("service should be provisioned")
	fmt.Println(cmd.Process, cmd.ProcessState, cmd.Dir, cmd.Env)
	fmt.Println("processid? ", cmd.Process.Pid)
	serviceState := &ServiceState{
		port:   cmd.Process.Pid,
		status: "ON",
		pid:    cmd.Process.Pid,
	}
	agent.services[instanceID] = serviceState
}

func (agent *MinioServiceAgent) DeleteInstanceHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	instanceID := vars["instance-id"]
	if _, found := agent.services[instanceID]; !found {
		agent.log.Error("instance not found", errors.New("instance does not exist"))
	}
	cmd := exec.Command("kill", "-9", strconv.Itoa(agent.services[instanceID].pid))

	f, err := os.Open("/tmp/log")
	if err != nil {
		agent.log.Fatal("Failed to deprovision an instance", err)
	}
	cmd.Stdout = f
	cmd.Stderr = f
	// Execute command

	err = cmd.Start() // will not wait for command to return

	// Only output the commands stdout

	if err != nil {
		agent.log.Fatal("Failed to deprovision instance", err)
	}
	fmt.Println("service should be deprovisioned")
	delete(agent.services, instanceID)
}
func (agent *MinioServiceAgent) InstanceStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("InstanceStatusHandler!\n"))
}
