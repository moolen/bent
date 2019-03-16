package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

var (
	targets            []string
	headersToPropagate = []string{
		"x-ot-span-context",
		"x-Request-id",
		"x-b3-traceid",
		"x-b3-spanid",
		"x-b3-parentspanid",
		"x-b3-sampled",
		"x-b3-flags",
		"uber-trace-id",
	}
)

func main() {
	targets = parseTargets()
	http.HandleFunc("/", httpHandler)
	http.HandleFunc("/healthz", healthz)
	http.ListenAndServe(":3000", http.DefaultServeMux)
}

// parseTargets parses a list of targets from a comma-separated string
func parseTargets() []string {
	var targets []string
	targetFromEnv := os.Getenv("TARGET")
	if targetFromEnv == "" {
		log.Fatal("missing ${TARGET}")
	}
	splitTargets := strings.Split(targetFromEnv, ",")
	for _, target := range splitTargets {
		targets = append(targets, target)
	}
	return targets
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	for _, target := range targets {
		log.Printf("issuing request to: %s", target)
		err = doRequest(r, target)
		// if request to the remote fails, we send an error
		if err != nil {
			log.Printf("error at request to %s: %s\n", target, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("received response from: %s", target)

	}

	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "\n========== your request ==========\n")
	w.Write(dump)
}

func doRequest(r *http.Request, target string) error {
	c := http.Client{}
	endpoint := fmt.Sprintf("%s", target)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	copyHeaders(r, req)
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

func copyHeaders(from, to *http.Request) {
	for _, key := range headersToPropagate {
		header := from.Header.Get(key)
		if header != "" {
			to.Header.Set(key, header)
			log.Printf("copying header %s : %s\n", key, header)
		}
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
