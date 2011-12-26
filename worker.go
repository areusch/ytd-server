package main

import(
	"exec"
	"fmt"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"strconv"
)

var binary = flag.String("downloader", "youtube-dl", "Path to youtube-dl")
var fileNameFormat = flag.String("name_format", "%(stitle)s.%(ext)s", "File name format to pass to youtube-dl")
var outputFormat = flag.String("output_format", "mp3", "Output format to write.")


type DownloadState int
const (
	kStarting DownloadState = 1
	kDownloadingMetadata = 2
	kDownloadingVideo = 3
	kExtractingAudio = 4
	kTagging = 5
	kComplete = 6
)

type DownloadRequest struct {
	id int
	url string
	basePath string
	statusChannel chan StatusMessage
}

func (d *DownloadRequest) WriteError(step string, e os.Error) {
	fmt.Printf("Error (%s): %v -> %v\n", step, e, d.statusChannel)
	var msg StatusMessage
	msg.requestId = d.id
	msg.step = step
	msg.error = e
	d.statusChannel <- msg
}

func MakeCommand(request *DownloadRequest, tempFile string) *exec.Cmd {
	bin, e := filepath.Abs(*binary)
	if e != nil {
		return nil
	}

	cmd := exec.Command(bin, "-o", tempFile, "-c", "-k", "--write-info-json", "--extract-audio", "--audio-format=mp3", "--audio-quality=192k", request.url)

	cmd.Dir = request.basePath

	return cmd
}

type StatusMessage struct {
	requestId int
	state DownloadState
	link string
	title string
	bytesTransferred, totalBytes, secondsRemaining uint64
	error os.Error
	step string
}

func (s StatusMessage) IsError() bool {
	return s.error != nil
}

func (s StatusMessage) IsLast() bool {
	return s.IsError() || s.state == kComplete
}

func SizeStringToInt(s string) (uint64, os.Error) {
	base, err := strconv.Atof64(s[0:len(s) - 1])
	if err != nil {
		return 0, err
	}

	switch strings.ToLower(s[len(s) - 1:len(s)])[0] {
	case 'k':
		return uint64(base * 1024), nil
	case 'm':
		return uint64(base * 1024 * 1024), nil
	case 'g':
		return uint64(base * 1024 * 1024 * 1024), nil
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return uint64(base * 10) + uint64((uint8(s[len(s) - 1]) - '0')), nil
	}
	return 0, os.NewError("Unknown trailing char in size string!")
}

func NewStatusMessage(s string) *StatusMessage {
	var msg StatusMessage
	progressRe := regexp.MustCompile("\\[download\\][ \t]+([0-9.]+)%[ \t]+of[ \t]+([0-9\\.mkgb]+) at[ \t]+([0-9.gkmb\\-]+)/s[ \t]+eta[ \t]+([0-9\\-]+):([0-9\\-]+)")
	if strings.HasPrefix(s, "[info]") && strings.Index(s, ":") != -1 {
		msg.state = kDownloadingMetadata
	} else if match := progressRe.FindStringSubmatch(strings.ToLower(s)); match != nil {
		msg.state = kDownloadingVideo
		progress, e := strconv.Atof32(match[1])
		if e != nil {
			return nil
		}
		msg.totalBytes, e = SizeStringToInt(match[2])
		if e != nil {
			return nil
		} else {
			minsRemaining, e := strconv.Atoi(match[4])
			if e == nil {
				secsRemaining, e := strconv.Atoi(match[5])
				if e != nil {
					msg.secondsRemaining = uint64(minsRemaining * 60 + secsRemaining)
				}
			}

			msg.bytesTransferred =
				uint64(float32(msg.totalBytes) * (progress / 100))
		}
	} else if strings.HasPrefix(s, "[ffmpeg]") && strings.Index(s, ":") != -1 {
		msg.state = kExtractingAudio
	}

	return &msg
}

type StdoutProcessor struct {
	readAhead string
	in *io.ReadCloser
	data []byte
	savedError os.Error
}

func NewStdoutProcessor(in *io.ReadCloser) *StdoutProcessor {
	return &StdoutProcessor{"", in, make([]byte, 256), nil}
}

func HasNewLineCR(s string) bool {
	return strings.IndexAny(s, "\r\n") != -1
}

func ExtractFirstToken(s string) (string, string) {
	n := strings.IndexAny(s, "\r\n")
	return s[0:n], s[n + 1:]
}

func (p *StdoutProcessor) ParseOneStatusMessage() (*StatusMessage, os.Error) {
	if HasNewLineCR(p.readAhead) {
		chunk, rest := ExtractFirstToken(p.readAhead)
		p.readAhead = rest
		return NewStatusMessage(chunk), nil
	} else {
		// Need to read more data from in. But first check savedError.
		if p.savedError != nil {
			return nil, p.savedError
		}

		n, e := (*(p.in)).Read(p.data)
		if n > 0 {
			p.readAhead = p.readAhead + string(p.data[0:n])
		}

		if HasNewLineCR(p.readAhead) {
			p.savedError = e
			chunk, rest := ExtractFirstToken(p.readAhead)
			p.readAhead = rest
			return NewStatusMessage(chunk), nil
		}

		if e != nil {
			return nil, e
		}

		return nil, os.NewError("No more data")
	}
	return nil, os.NewError("should not get here")
}

func ProcessRequest(request DownloadRequest) {
	var tempFile string = MakeTempFileName(request.url, ".flv")
	var tempFileSansExtension string
	{
		pathDotIndex := strings.LastIndex(tempFile, ".")
		tempFileSansExtension = tempFile[0:pathDotIndex]
	}

	var cmd *exec.Cmd = MakeCommand(&request, tempFile)

	if cmd == nil {
		request.WriteError("forming the download command", nil)
		return
	}

	var proc *StdoutProcessor
	if pipe, e := cmd.StdoutPipe(); e != nil {
		request.WriteError("setting youtube-dl pipe", e)
		return
	} else {
		proc = NewStdoutProcessor(&pipe)
	}

	if e := cmd.Start(); e != nil {
		request.WriteError("running youtube-dl", e)
		return
	}

	var state DownloadState = kStarting
	var data map[string] interface{}
	var e os.Error

	for {
		status, e := proc.ParseOneStatusMessage()
		if e != nil {
			if state < kExtractingAudio {
				request.WriteError("downloading the video", e)
				return
			} else {
				break
			}
		} else if status == nil {
			continue
		}

		if status.state != kDownloadingVideo && (status.state == 0 || status.state == state) {
			continue
		}

		fmt.Printf("Proc SM: %v\n", status)
		if status.state >= kDownloadingVideo && state != status.state {
			if data, e = ReadJson(tempFile + ".info.json"); e != nil {
				request.WriteError("reading video JSON", e)
				return
			}
			status.title = data["title"].(string)
		}
		state = status.state

		status.requestId = request.id
		request.statusChannel <- *status
	}

	var finalFileBaseName = data["title"].(string) + "." + *outputFormat
	var finalFile = request.basePath + "/" + finalFileBaseName
	if _, e := CopyFile(finalFile,
		tempFileSansExtension + "." + *outputFormat); e != nil {
		request.WriteError("tagging", e)
		return
	}

	request.statusChannel <- StatusMessage{request.id, kTagging, "", "", 0, 0, 0, nil, ""}
	var c = exec.Command("lltag", "--yes", "-G", "--rename=%a/%t", finalFileBaseName)
	c.Dir = request.basePath
	out, outError := c.StdoutPipe()
	if outError != nil {
		request.WriteError("starting the tagger", outError)
		return
	}

	var link string
	if e = c.Start(); e != nil {
		request.WriteError("tagging", e)
		return
	} else {
		outData, e := ioutil.ReadAll(out)
		if e != nil {
			request.WriteError("reading the tagger output", e)
			return
		}
		match := regexp.MustCompile("New filename is '(.*)'\n[ \t]*[^C \t]").FindStringSubmatch(string(outData))
		fmt.Printf("out data: %s\n\nmatch: %v\n", string(outData), match)
		if match == nil {
			request.WriteError("finding the output file", os.NewError("fail"))
			return
		}

		link = "/files/" + match[1]
	}

	fmt.Printf("Link: %s\n", link)
	request.statusChannel <- StatusMessage{request.id, kComplete, link, "", 0, 0, 0, nil, ""}
 }
