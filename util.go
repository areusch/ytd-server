package main

import(
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"json"
	"os"
)

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
