package main

import (
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
)

var log = lager.NewLogger("minio-serviceagent")

func main() {
	//creds := auth.CredentialsV4{"miniobroker", "miniobroker123", "us-east-1"}

	// Create logger
	log.RegisterSink(lager.NewWriterSink(os.Stderr, lager.DEBUG))
	log.RegisterSink(lager.NewWriterSink(os.Stderr, lager.INFO))
	// Setup the agent
	agent := &MinioServiceAgent{
		log:      log,
		rootURL:  "http://127.0.0.1",
		services: make(map[string]*ServiceState, 10),
	}
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "9001"
	}
	r := mux.NewRouter().SkipClean(true)
	// API Router
	apiRouter := r.NewRoute().PathPrefix("/").Subrouter()

	// Instance router
	instance := apiRouter.PathPrefix("/instance/{instance-id}").Subrouter()

	// Instanceprovision
	instance.Methods("PUT").HandlerFunc(agent.CreateInstanceHandler)
	instance.Methods("DELETE").HandlerFunc(agent.DeleteInstanceHandler)

	// PutObjectPart
	r.HandleFunc("/instance/{key}", agent.CreateInstanceHandler)
	r.HandleFunc("/instance/status", agent.InstanceStatusHandler)
	//r.HandleFunc("/binding/create", CreateBindingHandler)
	//r.HandleFunc("/binding/delete", DeleteBindingHandler)
	//r.HandleFunc("/binding/status", BindingStatusHandler)
	http.ListenAndServe(":9001", r)
}
