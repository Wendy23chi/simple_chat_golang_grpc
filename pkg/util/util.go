package util

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var (
	reader = bufio.NewReader(os.Stdin)
)

// RequestInput requests user input in string
func RequestInput(title string) string {
	for {
		fmt.Print(title + " : ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		return input
	}
}

// EncryptString encrypts a string using SHA256
func EncryptString(text string) string {
	sha := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sha[:])
}

// GetJSON reads JSON in storage
func GetJSON(filename string) []byte {
	// Open jsonFile
	jsonFile, err := os.Open("storage/" + filename)
	if err != nil {
		fmt.Println(err)
	}

	// Defer the closing of jsonFile so that it can parsed later on
	defer jsonFile.Close()

	// Read opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	return byteValue
}

// CreateLine creates a line in the console
func CreateLine() {
	fmt.Println("#################################################")
}
