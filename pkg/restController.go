package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jszwec/csvutil"
)

type JobPool struct{ Jobs map[string]*MigrationJob }

var jobPool JobPool

func init() {
	jobPool = JobPool{
		Jobs: map[string]*MigrationJob{},
	}
}

func RunServer() {
	http.HandleFunc("/tenants/migration", migrationHandler)
	http.HandleFunc("/jobs/status", jobStatusHandler)
	http.HandleFunc("/logs", logFetchHandler)
	port := "8085"
	Logger.Info("HTTP Server started; Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func logFetchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respFailure(w, http.StatusMethodNotAllowed, "Only GET HTTP Method supported for this endpoint")
		return
	}

	regLog, errLog, err := FetchFileLogs()
	if err != nil {
		respFailure(w, http.StatusInternalServerError, "Error while fetching log files: "+err.Error())
		return
	}

	var sb strings.Builder
	sb.WriteString("Error Log:\n")
	sb.WriteString("------------------------\n")
	sb.WriteString(string(errLog) + "\n")
	sb.WriteString("\n")
	sb.WriteString("\n")
	sb.WriteString("Runtime Log:\n")
	sb.WriteString("------------------------\n")
	sb.WriteString(string(regLog))

	w.Write([]byte(sb.String()))
}

func jobStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respFailure(w, http.StatusMethodNotAllowed, "Only GET HTTP Method supported for this endpoint")
		return
	}
	isSingleJobRequest := false
	jobIds, ok := r.URL.Query()["id"]
	if ok {
		isSingleJobRequest = true
	}

	if isSingleJobRequest {
		jobId := jobIds[0]
		job, exists := jobPool.Jobs[string(jobId)]
		if !exists {
			respFailure(w, http.StatusBadRequest, fmt.Sprintf("Could not find job for ID '%s'", jobId))
			return
		}

		jsonResp, err := json.Marshal(job)
		if err != nil {
			respFailure(w, http.StatusInternalServerError, "Error while marshalling Job")
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
		return
	}

	jsonResp, err := json.Marshal(jobPool)
	if err != nil {
		respFailure(w, http.StatusInternalServerError, "Error while marshalling Job")
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}

func migrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respFailure(w, http.StatusMethodNotAllowed, "Only POST HTTP Method supported")
		return
	}
	if r.Body == nil {
		respFailure(w, http.StatusBadRequest, "No body provided in the request")
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if len(body) == 0 {
		respFailure(w, http.StatusBadRequest, "Body is empty")
		return
	}
	if err != nil {
		respFailure(w, http.StatusBadRequest, "Error while reading body: "+err.Error())
		return
	}
	var tenants []Tenant
	if err := csvutil.Unmarshal(body, &tenants); err != nil {
		respFailure(w, http.StatusBadRequest, "Error while marshalling tenant objects => "+err.Error())
		return
	}

	job := MigrationJob{
		Id:        RandomString(8),
		StartTime: time.Now(),
		Status:    "INITIALIZED",
		ErrorLog:  make([]map[string]string, 0),
		WarnLog:   make([]map[string]string, 0),
		Audit:     make(map[string]*MigrationAudit),
		Tenants:   tenants,
	}
	jobPool.Jobs[job.Id] = &job
	go BulkMigration(&job)

	respSuccess(w, http.StatusCreated, "Migration started", job.Id)
}

func respSuccess(w http.ResponseWriter, statuscode int, message string, jobId string) {
	w.WriteHeader(statuscode)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = message
	resp["jobId"] = jobId
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}

func respFailure(w http.ResponseWriter, statuscode int, message string) {
	w.WriteHeader(statuscode)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["status"] = fmt.Sprint(statuscode)
	resp["message"] = message
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}
