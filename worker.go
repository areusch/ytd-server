package main

import(
	"exec"
	"fmt"
	"flag"
	"http"
	"hash/crc32"
	"io"
	"io/ioutil"
	"json"
	"os"
	"path/filepath"
	"strings"
)

var binary = flag.String("downloader", "youtube-dl", "Path to youtube-dl")
var fileNameFormat = flag.String("name_format", "%(stitle)s.%(ext)s", "File name format to pass to youtube-dl")
var outputFormat = flag.String("output_format", "mp3", "Output format to write.")

type DownloadRequest struct {
	url string
	basePath string
}

func CopyFile(dst, src string) (int64, os.Error) {
        sf, err := os.Open(src)
        if err != nil {
                return 0, err
        }
        defer sf.Close()
        df, err := os.Create(dst)
        if err != nil {
                return 0, err
        }
        defer df.Close()
        return io.Copy(df, sf)
}

func MakeTempFileName(fileName, suffix string) string {
	return fmt.Sprintf("%s/tmp.%x%s", os.TempDir(), crc32.ChecksumIEEE([]byte(fileName)), suffix)
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

func ReadJson(fileName string) (map[string] interface{}, os.Error) {
	var file []byte
	var e os.Error

	if file, e = ioutil.ReadFile(fileName); e != nil {
		return nil, e
	}

	var data map[string] interface{}
	if e := json.Unmarshal(file, &data); e != nil {
		return nil, e
	}

	return data, nil
}

func ProcessRequest(request DownloadRequest, w http.ResponseWriter, r *http.Request) {
	var tempFile string = MakeTempFileName(request.url, ".flv")
	var cmd *exec.Cmd = MakeCommand(&request, tempFile)

	if cmd == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Could not form command!"))
		return
	}

	if e := cmd.Run(); e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if cmd.Stdin == nil {
			w.Write([]byte("An exception occurred while running yt-downloader:\n"))
			w.Write([]byte(e.String()))
			return
		}
		if data, e := ioutil.ReadAll(cmd.Stdin); e != nil {
			w.Write([]byte("No data was given."))
		} else {
			w.Write([]byte("Stack Trace is as follows:"))
			w.Write(data)
		}
		return
	}

	var data map[string] interface{}
	var e os.Error
	if data, e = ReadJson(tempFile + ".info.json"); e != nil {
		w.WriteHeader(500)
		w.Write([]byte(e.String()))
		return
	}

	pathDotIndex := strings.LastIndex(tempFile, ".")
	var finalFile = request.basePath + "/" + data["title"].(string) + "." + *outputFormat
	if _, e := CopyFile(finalFile,
		tempFile[0:pathDotIndex] + "." + *outputFormat); e != nil {
		w.WriteHeader(500)
		w.Write([]byte(e.String()))
		return
	}

	var c = exec.Command("lltag", "--yes", finalFile)
	if e = c.Run(); e != nil {
		w.WriteHeader(500)
		w.Write([]byte(e.String()))
		return
	}
	w.WriteHeader(200)
	w.Write([]byte("{\"encoded\": \"" + data["title"].(string) + "\"}"))
}
