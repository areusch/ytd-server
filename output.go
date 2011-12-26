package main

import(
	"fmt"
	"http"
	"json"
	"os"
)

type MessageType int
const (
	_ = iota
	MessageTypeStatus MessageType = 1
	MessageTypeError MessageType = 2
)

type OutputWriter interface {
	WriteError(step string, e os.Error)
	WriteStatus(msg StatusMessage)
}

type PlainOutputWriter struct {
	requestId int
	w *http.ResponseWriter
}

func (p *PlainOutputWriter) WriteStatus(msg StatusMessage) {
	(*(p.w)).Write([]byte(fmt.Sprintf("STATUS (%d): %d (%d/%d)\n", msg.requestId, msg.state, msg.bytesTransferred, msg.totalBytes)))
	(*(p.w)).(http.Flusher).Flush()
}

func (p *PlainOutputWriter) WriteError(step string, e os.Error) {
	(*(p.w)).WriteHeader(500)
	(*(p.w)).Write([]byte("While " + step + ": "))
	if e != nil {
		(*(p.w)).Write([]byte(e.String()))
	} else {
		(*(p.w)).Write([]byte("No error string given!"))
	}

	(*(p.w)).(http.Flusher).Flush()
}


type JsonPOutputWriter struct {
	PlainOutputWriter
//	jsonPCallback string
}

func (j *JsonPOutputWriter) WriteJson(m map[string] string) {
	(*(j.w)).Header().Add("Content-Type", "application/json")
	m["requestId"] = fmt.Sprintf("%d", j.requestId)
	encoded, e := json.MarshalForHTML(m)
	if e != nil {
		(*(j.w)).Write([]byte("An error occurred while marshalling the following JSON:"))
		(*(j.w)).Write([]byte(e.String()))
		return
	}

//	(*(j.w)).Write([]byte("<script type=\"text/javascript\">" + j.jsonPCallback + "("))
	(*(j.w)).Write(encoded)
//	(*(j.w)).Write([]byte(");</script>\n"))
}

func (j *JsonPOutputWriter) WriteError(step string, e os.Error) {
	m := make(map[string] string)
	m["type"] = fmt.Sprintf("%d", MessageTypeError)
	m["step"] = step
	if e != nil {
		m["error"] = e.String()
	} else {
		m["error"] = ""
	}
	j.WriteJson(m)
}

func (j *JsonPOutputWriter) WriteStatus(msg StatusMessage) {
	m := make(map[string] string)
	m["state"] = fmt.Sprintf("%d", msg.state)
	m["link"] = msg.link
	m["title"] = msg.title
	m["secondsRemaining"] = fmt.Sprintf("%d", msg.secondsRemaining)
	m["bytesTransferred"] = fmt.Sprintf("%d", msg.bytesTransferred)
	m["totalBytes"] = fmt.Sprintf("%d", msg.totalBytes)
	if msg.error != nil {
		m["error"] = msg.error.String()
		m["step"] = msg.step
	}
	j.WriteJson(m)
	(*(j.w)).(http.Flusher).Flush()
}
