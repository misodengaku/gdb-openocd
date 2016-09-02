package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	pipe "gopkg.in/pipe.v2"
)

type Config struct {
	GDBPath       string `json:"gdb_path"`
	OpenOCDOption string `json:"openocd_option"`
	OpenOCDPath   string `json:"openocd_path"`
}

func main() {
	file, err := ioutil.ReadFile("gdb_wrapper.json")
	if err != nil {
		fmt.Println("cat't load gdb_wrapper.json")
		os.Exit(1)
	}

	config := Config{}

	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("gdb_wrapper.json Unmarshal error")
		panic(err)
	}

	openocdKillChan := make(chan struct{})
	fmt.Println("OpenOCD")
	openocdParams := strings.Split(config.OpenOCDOption, " ")
	//openocdParams := []string{"-s", "c:\\openocd\\tcl", "-f", "interface/stlink-v2.cfg", "-f", "target/stm32l4x.cfg"}
	openocdCmd := exec.Command(config.OpenOCDPath, openocdParams...)
	err = openocdCmd.Start()
	if err != nil {
		fmt.Println("OpenOCD launch fail")
		panic(err)
	}
	defer openocdCmd.Process.Kill()

	go func() {
		fmt.Println("OpenOCD PID:", openocdCmd.Process.Pid)
		// Target voltage
		<-openocdKillChan
		openocdCmd.Process.Kill()
		fmt.Println("OpenOCD killed.")
	}()

	gdbArgs := make([]string, len(os.Args)-1)
	for i, v := range os.Args {
		if i == 0 {
			continue
		}
		gdbArgs[i-1] = v
	}

	gdbCmd := exec.Command(config.GDBPath, gdbArgs...)

	gdbStdin, err := gdbCmd.StdinPipe()
	gdbStdout, err := gdbCmd.StdoutPipe()

	err = gdbCmd.Start()
	fmt.Println("gdb started")
	if err != nil {
		fmt.Println("gdb")
		panic(err)
	}

	go func() {
		for {
			stdinPipe := pipe.Line(
				pipe.Read(os.Stdin),
				pipe.Write(gdbStdin),
			)

			if err = pipe.Run(stdinPipe); err != nil {
				openocdKillChan <- struct{}{}
				panic(err)
			}
			if s, err := pipe.Output(stdinPipe); err != nil {
				fmt.Println(s)
				openocdKillChan <- struct{}{}
				panic(err)
			}
		}
	}()

	go func() {
		for {
			stdoutPipe := pipe.Line(
				pipe.Read(gdbStdout),
				pipe.Write(os.Stdout),
			)
			if err = pipe.Run(stdoutPipe); err != nil {
				openocdKillChan <- struct{}{}
				panic(err)
			}
			if s, err := pipe.Output(stdoutPipe); err != nil {
				fmt.Println(s)
				openocdKillChan <- struct{}{}
				panic(err)
			}
		}
	}()
	gdbCmd.Wait()
	openocdKillChan <- struct{}{}
}
