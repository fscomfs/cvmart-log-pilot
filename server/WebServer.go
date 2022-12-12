package server

import (
	"fmt"
	"net/http"
)

func LogtHandler() {
	http.HandleFunc("/log", RequestHandler)
	http.ListenAndServe(":888", nil)
	fmt.Print("ListenAndServe")
}
