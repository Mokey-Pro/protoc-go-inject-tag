package main

import (
	"flag"
	"github.com/gogf/gf/text/gstr"
	"github.com/golang/glog"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
)

func main() {
	var wholeInputFile string
	//var xxxTags string
	flag.StringVar(&wholeInputFile, "input", "", "path to input file")
	//flag.StringVar(&xxxTags, "XXX_skip", "", "skip tags to inject on XXX fields")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")

	flag.Parse()

	inputFiles := parseFilePath(wholeInputFile)

	for _, each := range inputFiles {
		handleProto(each)
	}
}

// 解析带*号的文件路径
// filePath = `/user/*.proto`, 查找/user文件夹下所有的.proto文件
func parseFilePath(filePath string) []string {
	if !gstr.Contains(filePath, "*") {
		return []string{filePath}
	}

	idx := strings.LastIndex(filePath, "/")
	prefixPath := gstr.SubStr(filePath, 0, idx+1)
	likeFileName := gstr.SubStr(filePath, idx+1, -1)
	glog.Infof("prefixPath = %v,likeFileName=%v", prefixPath, likeFileName)

	regStr := strings.Replace(likeFileName, "*", `.*`, -1)
	reg, _ := regexp.Compile(`^` + regStr + `$`)

	files, _ := ioutil.ReadDir(prefixPath)

	retList := make([]string, 0)

	for _, eachFile := range files {
		if ok := reg.MatchString(eachFile.Name()); ok {
			retList = append(retList, prefixPath+eachFile.Name())
		}
	}

	return retList
}

// 处理proto文件
func handleProto(inputFile string) {
	if len(inputFile) == 0 {
		log.Fatal("input file is mandatory")
	}

	areasMap, err := parseFile(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	if err = writeFile(inputFile, areasMap); err != nil {
		log.Fatal(err)
	}
}
