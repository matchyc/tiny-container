package main

/* Created by Meng Chen
 * Build a container from scratch
 * Expect to build a container which could run all bash commands inside
 * E-mail: Meng_chen@bupt.edu.cn
 */

import (
	"fmt"
	"os"
	"os/exec"
)

func run() {
	//The second argument and the successive args are the targer command needed to run
	fmt.Printf("Runnning comand: %v\n", os.Args[2:])

	//Golang package exec runs external commands. (Like exec in C?)
	//os/exec package do not invoke any system shell or something like pipe and so on.
	//it just like 'exec' in C
	//Type cmd and func Command returns the Cmd struct to execute the named program
	//with given arguments. (In this case, os.Args[2] is the program name)
	cmd := exec.Command(os.Args[2], os.Args[3:]...) // use '...' to split args.

	//golang os/exec wrapped os.StartProcess to make it easier to remap stdin and stdout.
	//eg. we can remap easily like
	//cmd.Stdin = strings.NewReader("input example")
	//but we don't need to remap in this situation
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//just run
	err := cmd.Run()
	if err != nil {
		panic(fmt.Sprintf("Running error: %v\n", err))
	}
}

func main() {
	switch os.Args[1] {
	//assume "run" is the first argument for start.
	case "run":
		run()
	default:
		//if the first argument is not "run", then trigger a panic
		panic("BAD COMMAND!!! (the first argument must be \"run\")")
	}
}
