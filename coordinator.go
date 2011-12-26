package main

import(
	"fmt"
	"os"
)

type CoordinatorMessage struct {
	id int
	status StatusMessage
	newRequest OngoingRequest
	link string
	outChan chan CoordinatorMessage
}

type OngoingRequest struct {
	request DownloadRequest
	link string
	messages chan StatusMessage
}

type DownloadCoordinator struct {
	ongoingDownloads map[int] OngoingRequest
	completedDownloads map[int] string
	in chan CoordinatorMessage
	status chan StatusMessage
	nextRequestId int
}

func NewDownloadCoordinator() (*DownloadCoordinator) {
	coord := &DownloadCoordinator{make(map[int] OngoingRequest),
		make(map[int] string),
		make(chan CoordinatorMessage),
		make(chan StatusMessage),
		0}
	go coord.Coordinate()
	return coord
}

func (d *DownloadCoordinator) Coordinate() {
	for {
		fmt.Printf("\n\nCoord %v.\n", d.status)
		select {
		case req := <- d.in:
			fmt.Printf("Coord req %v\n", req);
			ProcessOneCoordinationRequest(d, req)
		case msg := <- d.status:
			fmt.Printf("Coord status %v\n", msg);
			ProcessOneStatusMessage(d, msg)
		}
	}
}

func ProcessOneStatusMessage(d *DownloadCoordinator, msg StatusMessage) {
	if download, ok := d.ongoingDownloads[msg.requestId]; ok {
		if msg.link != "" {
			download.link = msg.link
		}

		fmt.Printf("Status message\n")
		if msg.IsLast() {
			d.ongoingDownloads[msg.requestId] = download, false
			d.completedDownloads[msg.requestId] = download.link
		}
		fmt.Printf("MQ(%d): Writing %v to %v: ", msg.requestId, msg, download.messages)
		os.Stdout.Sync()
		for messageWritten := false; !messageWritten; {
			select {
			case download.messages <- msg:
				fmt.Printf("\nWrote to queue.\n")
				messageWritten = true
			default:
				e := <- download.messages
				fmt.Printf("(spill %v for %v) ", e, msg)
				os.Stdout.Sync()
			}
		}
		if msg.IsLast() {
			fmt.Printf("closing\n");
			close(download.messages)
		}
	}
	fmt.Printf("POSM exiting\n")
}

func (d *DownloadCoordinator) NewDownloadRequest(req CoordinatorMessage) OngoingRequest {
	d.nextRequestId++
	return OngoingRequest{DownloadRequest{d.nextRequestId, "", "", make(chan StatusMessage, 1)}, "", make(chan StatusMessage, 30)}
}

func PullNextStatusMessage(msg CoordinatorMessage, ongoing OngoingRequest, outChan chan CoordinatorMessage) {
	msg.status = <- ongoing.messages
	fmt.Printf("Got msg: %v\n", msg.status)
	outChan <- msg
}

func ProcessOneCoordinationRequest(d *DownloadCoordinator, req CoordinatorMessage) {
	var msg CoordinatorMessage
	if req.id == 0 {
		msg.newRequest = d.NewDownloadRequest(req)
		fmt.Printf("Create request %v\n", msg.newRequest.messages);
		msg.id = d.nextRequestId
		d.ongoingDownloads[msg.id] = msg.newRequest
	} else if ongoing, ok := d.ongoingDownloads[req.id]; ok {
		fmt.Printf("Existing request: %v\n", ongoing)
		msg.id = req.id
		go PullNextStatusMessage(msg, ongoing, req.outChan)
		return
	} else if link, ok := d.completedDownloads[req.id]; ok {
		fmt.Printf("Completed Request: %s\n", link);
		msg.id = req.id
		msg.status.state = kComplete
		msg.status.link = link
	}
	req.outChan <- msg
}
