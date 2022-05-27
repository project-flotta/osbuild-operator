package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/u-root/u-root/pkg/shlex"
)

var hexEscape = regexp.MustCompile(`\\x[0-9a-fa-f]{2}`)

func AppendKickstartTogrub(grubcfgPath string, kickstartPath string) (*bytes.Buffer, error) {

	var file, err = os.Open(grubcfgPath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	output := &bytes.Buffer{}
	writer := bufio.NewWriter(output)

	for scanner.Scan() {
		line := hexEscape.ReplaceAllString(scanner.Text(), `\\$0`)
		kv := shlex.Argv(line)
		if len(kv) == 0 {
			// It's an empty line, just keep it
			_, err = writer.WriteString(fmt.Sprintf("%s\n", line))
			if err != nil {
				WarningLogger.Printf("cannot write grub file, line: %s", line)
			}
			continue
		}
		directive := strings.ToLower(kv[0])
		if directive != "linuxefi" {
			_, err = writer.WriteString(fmt.Sprintf("%s\n", line))
			if err != nil {
				WarningLogger.Printf("cannot write grub file, line: %s", line)
			}
			continue
		}

		_, err = writer.WriteString(fmt.Sprintf("%s inst.ks=cdrom:/%s\n", line, kickstartPath))
		if err != nil {
			WarningLogger.Printf("cannot append kickstart file to grub: %s", line)
		}
	}
	writer.Flush()
	return output, nil
}

func AppendKickstartToIsoLinux(isolinuxPath string, kickstartPath string) (*bytes.Buffer, error) {

	var file, err = os.Open(isolinuxPath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	output := &bytes.Buffer{}
	writer := bufio.NewWriter(output)
	for scanner.Scan() {
		line := scanner.Text()
		kv := strings.Fields(line)
		if len(kv) <= 1 {
			_, err = writer.WriteString(fmt.Sprintf("%s\n", line))
			if err != nil {
				WarningLogger.Printf("cannot write isolinux file, line: %s", line)
			}
			continue
		}

		directive := strings.ToLower(kv[0])
		if directive != "append" {
			_, err = writer.WriteString(fmt.Sprintf("%s\n", line))
			if err != nil {
				WarningLogger.Printf("cannot write isolinux file, line: %s", line)
			}
			continue
		}

		_, err = writer.WriteString(fmt.Sprintf("%s inst.ks=cdrom:/%s\n", line, kickstartPath))
		if err != nil {
			WarningLogger.Printf("cannot append kickstart file to isolinux: %s", line)
		}
	}
	writer.Flush()
	return output, nil
}
