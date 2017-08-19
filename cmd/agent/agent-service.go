package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/minio/minio-service-broker/utils"
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
	_, err := exec.LookPath("minio")
	if err != nil {
		agent.log.Info("minio binary not found in install paths")
	}
	port := getFreePort()

	// minio directory path
	dirPath := RootDir + instanceID + "/" + "data"
	configDirPath := getConfigDir(instanceID)
	fmt.Println(configDirPath, "diroath")
	cmd := exec.Command("minio", "server", "--address", ":"+strconv.Itoa(port), "--config-dir", configDirPath, dirPath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start() // will wait for command to return

	// Only output the commands stdout

	if err != nil {
		agent.log.Fatal("Failed to provision instance", err)
	}
	fmt.Println("Service provisioned successfully")
	serviceState := &ServiceState{
		port:   port,
		status: "ON",
		pid:    cmd.Process.Pid,
	}
	agent.services[instanceID] = serviceState
	dashboardURL := agent.getDashboardURL(instanceID)
	w.Header().Set("Content-Type", string("application/txt"))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, dashboardURL)

}

func (agent *MinioServiceAgent) DeleteInstanceHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	instanceID := vars["instance-id"]
	if _, found := agent.services[instanceID]; !found {
		agent.log.Error("instance not found", errors.New("instance does not exist"))
	}
	cmd := exec.Command("kill", "-9", strconv.Itoa(agent.services[instanceID].pid))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Execute command

	if err := cmd.Run(); err != nil {
		agent.log.Fatal("Failed to deprovision instance", err)
	}
	fmt.Println("Service should be deprovisioned")
	delete(agent.services, instanceID)
	w.WriteHeader(http.StatusOK)

}
func (agent *MinioServiceAgent) InstanceStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("InstanceStatusHandler!\n"))
}

func (agent *MinioServiceAgent) GetInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("Entering GetInstanceHandler handler ...")
	vars := mux.Vars(r)
	instanceID := vars["instance-id"]

	if _, found := agent.services[instanceID]; !found {
		agent.log.Error("instance not found", errors.New("instance does not exist"))
	}
	// load credentials from config
	configFilePath := getConfigFilePath(instanceID)
	creds, err := utils.GetCredentialsFromConfig(configFilePath)
	if err != nil {
		agent.log.Fatal("Instance config missing", err)
	}
	instanceURL := agent.rootURL + ":" + strconv.Itoa(agent.services[instanceID].port) + "/minio"
	credentials := &utils.Credentials{
		EndpointURL: instanceURL,
		AccessKey:   creds.AccessKey,
		SecretKey:   creds.SecretKey,
		Region:      "us-east-1",
	}
	// Marshal API response
	jsonBytes, err := json.Marshal(credentials)
	if err != nil {
		http.Error(w, "Credentials could not be marshalled to JSON", http.StatusInternalServerError)
		agent.log.Fatal("Failed to marshal instance credentials to json", err)
	}

	w.Header().Set("Content-Type", string("application/json"))
	w.WriteHeader(http.StatusOK)
	if credentials != nil {
		w.Write(jsonBytes)
		w.(http.Flusher).Flush()
	}
}

func getConfigDir(instanceID string) string {
	return RootDir + instanceID + "/" + "config"
}
func getConfigFilePath(instanceID string) string {
	return getConfigDir(instanceID) + "/config.json"
}
func (agent *MinioServiceAgent) getDashboardURL(instanceID string) string {
	instanceURL := agent.rootURL + ":" + strconv.Itoa(agent.services[instanceID].port) + "/minio"
	return instanceURL
}
