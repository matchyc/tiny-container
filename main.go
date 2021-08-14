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
	"syscall"
)

//first time
func run() {
	//The second argument and the successive args are the targer command needed to run
	fmt.Printf("Runnning command: %v as %d\n", os.Args[2:], os.Getpid())

	//Program 1
	//Golang package exec runs external commands. (Like exec in C?)
	//os/exec package do not invoke any system shell or something like pipe and so on.
	//it just like 'exec' in C
	//Type cmd and func Command returns the Cmd struct to execute the named program
	//with given arguments. (In this case, os.Args[2] is the program name)

	// cmd := exec.Command(os.Args[2], os.Args[3:]...) // use '...' to split args.

	//Program 2
	args := os.Args
	args[0] = "/proc/self/exe" //The file always links to the currenly executable
	args[1] = "child"
	cmd := exec.Command(args[0], args[1:]...)

	//NEWUST isolates system identifier like hostname...
	//NEWPID isolates process id number
	//
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
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

//second time, actually run the expected command
func child() {
	fmt.Printf("Running %v as %d\n", os.Args[2:], os.Getpid())

	// fmt.Println("Check executable path", cmd.Path)
	//set hostname in every command we executed
	err := syscall.Sethostname([]byte("tiny-container"))
	if err != nil {
		panic(fmt.Sprintf("Sethostname: %v\n", err))
	}

	//chroot
	err = syscall.Chroot("/home/meng/projects/tiny-container/alpine")
	if err != nil {
		panic(fmt.Sprintf("Chroot: %v\n", err))
	}

	//chdir, change to new root directory
	err = syscall.Chdir("/")
	if err != nil {
		panic(fmt.Sprintf("Chdir: %v\n", err))
	}
	// syscall.Setenv("PATH", os.Getenv("PATH"))
	// fmt.Println(os.Getenv("PATH"))

	err = syscall.Mount("proc", "proc", "proc", 0, "")
	if err != nil {
		panic(fmt.Sprintf("mount: %v\n", err))
	}

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//Ubuntu's executables file paths are different from alpine
	//So after chroot and chdir, we need to use LookPath to
	//get the program path and make cmd.Path equals progPath
	// progPath, err := exec.LookPath(os.Args[2])
	// cmd.Path = progPath

	err = cmd.Run()
	if err != nil {
		panic(fmt.Sprintf("running: %v\n", err))
	}

	err = syscall.Unmount("/proc", 0)
	if err != nil {
		panic(fmt.Sprintf("Umount: %v\n", err))
	}
}

func main() {
	switch os.Args[1] {
	//assume "run" is the first argument for start.
	case "run":
		run()
	case "child":
		child()
	default:
		//if the first argument is not "run", then trigger a panic
		panic("BAD COMMAND!!! (the first argument must be \"run\")")
	}
	fmt.Println("Golang program exited.")
}
