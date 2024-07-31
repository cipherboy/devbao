package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func Tail(path string, fullContext bool) error {
	fp, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for reading: %w", err)
	}

	defer fmt.Print("\n")
	defer fp.Close()

	data, err := io.ReadAll(fp)
	if len(data) > 0 {
		sData := string(data)
		if !fullContext {
			split := strings.Split(sData, "\n")
			idx := max(0, len(split)-15)
			split = split[idx:]
			sData = strings.Join(split, "\n")
		}

		fmt.Print(sData)
	}

	for {
		data, err := io.ReadAll(fp)
		if len(data) > 0 {
			fmt.Print(string(data))
		}

		if errors.Is(err, io.EOF) {
			// XXX - use bufio.ReadLine() and handle EOF explicitly.
		} else if err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}

		time.Sleep(25 * time.Millisecond)
	}
}
