package main

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
)

var log = lager.NewLogger("minio-serviceagent")

func main() {
	fmt.Println("agent listening...")

	// Create logger
	log.RegisterSink(lager.NewWriterSink(os.Stderr, lager.DEBUG))
	log.RegisterSink(lager.NewWriterSink(os.Stderr, lager.INFO))
	// Setup the agent
	agent := &MinioServiceAgent{
		log: log,
	}
	if err := agent.Init(); err != nil {
		log.Fatal("Unable to Init()", err)
		return
	}

	if err := os.MkdirAll(globalMinioDir, 0755); err != nil {
		log.Fatal("Unable to create "+globalMinioDir, err)
		return
	}
	if err := os.MkdirAll(globalInstancesDir, 0755); err != nil {
		log.Fatal("Unable to create "+globalInstancesDir, err)
		return
	}

	router := mux.NewRouter()
	router.Methods("PUT").Path("/instances/{instance-id}").HandlerFunc(agent.CreateInstanceHandler)
	router.Methods("DELETE").Path("/instances/{instance-id}").HandlerFunc(agent.DeleteInstanceHandler)
	router.Methods("GET").Path("/instances/{instance-id}").HandlerFunc(agent.GetInstanceHandler)

	if err := http.ListenAndServe(globalAgentPort, router); err != nil {
		log.Fatal("Unable to listen on port "+globalAgentPort, err)
	}
}
