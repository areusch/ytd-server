package main

import "fmt"
import "testing"

func TestStatusMessage(t *testing.T) {
	sm := NewStatusMessage("NSM: [download]  32.1% of 35.69M at  618.02k/s ETA 00:40")
	fmt.Printf("%v\n", sm);
	if sm.state != kDownloadingVideo {
		t.Fail()
	}
}
