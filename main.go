package main

import (
	"fmt"
	"os"
	"os/exec"

	pipe "gopkg.in/pipe.v2"
)

func main() {
	openocdKillChan := make(chan struct{})
	fmt.Println("OpenOCD")
	openocdParams := []string{"-s", "c:\\openocd\\tcl", "-f", "interface/stlink-v2.cfg", "-f", "target/stm32l4x.cfg"}
	openocdCmd := exec.Command("openocd", openocdParams...)
	openocdCmd.Start()
	defer openocdCmd.Process.Kill()

	go func() {
		fmt.Println("OpenOCD PID:", openocdCmd.Process.Pid)
		// Target voltage
		<-openocdKillChan
		openocdCmd.Process.Kill()
	}()

	gdbArgs := make([]string, len(os.Args)-1)
	for i, v := range os.Args {
		if i == 0 {
			continue
		}
		gdbArgs[i-1] = v
	}

	gdbCmd := exec.Command("C:\\gcc-arm\\bin\\arm-none-eabi-gdb.exe", gdbArgs...)

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
				panic(err)
			}
			if s, err := pipe.Output(stdinPipe); err != nil {
				fmt.Println(s)
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
				panic(err)
			}
			if s, err := pipe.Output(stdoutPipe); err != nil {
				fmt.Println(s)
				panic(err)
			}
		}
	}()
	gdbCmd.Wait()
}
