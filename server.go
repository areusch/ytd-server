package main

import(
	"http"
	"url"
)

var kKeyUrl = "url"

type DownloadHandler struct {
	basePath string
}

func NewDownloadHandler(basePath string) (*DownloadHandler) {
	var handler DownloadHandler
	handler.basePath = basePath
	return &handler
}

func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var _, e = url.Parse(r.FormValue(kKeyUrl))
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var request DownloadRequest
	request.url = r.FormValue(kKeyUrl)
	request.basePath = h.basePath
	ProcessRequest(request, w, r)
}
