package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func getUser(r *http.Request) string {
	if userName := r.Header.Get("from"); userName != "" {
		return userName
	}
	return "UNKNOWN"
}

func writeAsJSON(w http.ResponseWriter, statusCode int, obj interface{}) {
	jsonObj, err := json.Marshal(obj)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jsonObj)
}
