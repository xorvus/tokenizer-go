package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xorvus/tokenizer-go"
)

func main() {
	textList, modelList, encodingList := ReadTestFile()
	testTokenByModel(textList, modelList)
	fmt.Println("=========================================")
	testTokenByEncoding(textList, encodingList)
}

func ReadTestFile() (textList []string, modelList []string, encodingList []string) {
	file, err := os.Open("test.txt")
	if err != nil {
		file, err = os.Open("test/test.txt")
		if err != nil {
			log.Fatal(err)
		}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	textList = strings.Split(lines[0], ",")
	modelList = strings.Split(lines[1], ",")
	encodingList = strings.Split(lines[2], ",")

	return
}

func getTokenByModel(text string, model string) int {
	tkm, err := tokenizer.ForModel(model)
	if err != nil {
		log.Printf("ForModel error: %v", err)
		return 0
	}

	tokens, err := tkm.EncodeOrdinary(text)
	if err != nil {
		log.Printf("Encode error: %v", err)
		return 0
	}

	return len(tokens)
}

func getTokenByEncoding(text string, encoding string) int {
	tke, err := tokenizer.GetEncoding(tokenizer.Encoding(encoding))
	if err != nil {
		log.Printf("GetEncoding error: %v", err)
		return 0
	}

	tokens, err := tke.EncodeOrdinary(text)
	if err != nil {
		log.Printf("Encode error: %v", err)
		return 0
	}

	return len(tokens)
}

func testTokenByModel(textList []string, modelList []string) {
	for i := 0; i < len(textList); i++ {
		for j := 0; j < len(modelList); j++ {
			fmt.Printf("text: %s, model: %s, token: %d\n", textList[i], modelList[j], getTokenByModel(textList[i], modelList[j]))
		}
	}
}

func testTokenByEncoding(textList []string, encodingList []string) {
	for i := 0; i < len(textList); i++ {
		for j := 0; j < len(encodingList); j++ {
			fmt.Printf("text: %s, encoding: %s, token: %d\n", textList[i], encodingList[j], getTokenByEncoding(textList[i], encodingList[j]))
		}
	}
}
