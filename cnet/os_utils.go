package cnet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// loadData 载入数据
func loadData(filepath string) (data []byte, notExist bool, err error) {
	if _, err = os.Stat(filepath); err != nil {
		if os.IsNotExist(err) {
			err = nil
			notExist = true
		}
		return
	}
	data, err = ioutil.ReadFile(filepath)
	return
}

// loadJSON 加载 json 数据
func loadJSON(data interface{}, filepath string) (notExist bool, err error) {
	var byteData []byte
	if byteData, notExist, err = loadData(filepath); err != nil || notExist == true {
		return
	}
	if err = json.Unmarshal(byteData, data); err != nil {
		return false, fmt.Errorf("unmarshal data failed %v", err)
	}
	return
}

// dumpData 保存数据 (将会重写该文件内容)
func dumpData(data []byte, filepath string) (err error) {
	_ = os.Remove(filepath)
	err = ioutil.WriteFile(filepath, data, 0666)
	return
}

// dumpJSON 存储 json 数据
func dumpJSON(data interface{}, filepath string) (err error) {
	var byteData []byte
	if byteData, err = json.Marshal(data); err != nil {
		return fmt.Errorf("marshal json data fail %v, %v", data, err)
	}
	err = dumpData(byteData, filepath)
	return
}
