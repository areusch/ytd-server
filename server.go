package main

import(
	"fmt"
	"http"
	"strconv"
	"url"
)

var kKeyId = "id"
var kKeyUrl = "url"
var kJson = "json"

type DownloadHandler struct {
	basePath string
	coordinator *DownloadCoordinator
}

func NewDownloadHandler(basePath string, coordinator *DownloadCoordinator) (*DownloadHandler) {
	var handler DownloadHandler
	handler.basePath = basePath
	handler.coordinator = coordinator
	return &handler
}

func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var msg CoordinatorMessage
	var responseChan chan CoordinatorMessage = make(chan CoordinatorMessage, 1)

	idString := r.FormValue(kKeyId)
	isNewRequest := false
	if idString != "" {
		if id, e := strconv.Atoi(idString); e == nil {
			msg.id = id
		}
	} else {
		if _, e := url.Parse(r.FormValue(kKeyUrl)); e != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		isNewRequest = true
	}

	fmt.Printf("coordinate?\n")
	msg.outChan = responseChan
	h.coordinator.in <- msg
	response := <- responseChan
	fmt.Printf("DONE!\n")

	var out OutputWriter
	if cb := r.FormValue(kJson); cb != "" {
		out = &JsonPOutputWriter{PlainOutputWriter{response.id, &w}}
	} else {
		out = &PlainOutputWriter{response.id, &w}
	}

	out.WriteStatus(response.status)

	if isNewRequest {
		fmt.Printf("New request!\n");
		request := response.newRequest.request
		request.url = r.FormValue(kKeyUrl)
		request.basePath = h.basePath
		request.statusChannel = h.coordinator.status
		go ProcessRequest(request)
	}
}
