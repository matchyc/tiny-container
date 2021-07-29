# tiny-container
build a container from scratch in Golang

## PROGRAM 1
main.go could execute any commands like we are using bash.
**Usage**
```
go run main.go run ls -l
```
But if we try to run the command like `go run main.go run bash`, what will happen?

**Nothing happended and the go program do not exit.**(Cuase if exited, it will output "Exit."), Only if we use `exit`, then we can see the go prog exited.

What if we use `ps -af` to see what happen?

```s
meng@ali-ecs:~/projects/tiny-container$ ps -af
UID          PID    PPID  C STIME TTY          TIME CMD
meng       84933   84837  0 16:50 pts/2    00:00:00 ps -af
meng@ali-ecs:~/projects/tiny-container$ go run main.go run bash
Runnning comand: [bash]
meng@ali-ecs:~/projects/tiny-container$ ps -af
UID          PID    PPID  C STIME TTY          TIME CMD
meng       84934   84837  8 16:50 pts/2    00:00:00 go run main.go run bash
meng       84971   84934  0 16:50 pts/2    00:00:00 /tmp/go-build1114320235
meng       84975   84971  0 16:50 pts/2    00:00:00 bash
meng       84986   84975  0 16:50 pts/2    00:00:00 ps -af
```

Okay, now we see that.
- The first line is the command we just entered.
- /tmp/ is a directory to store go compiled executable.
   The program cloned a **new** process to run `bash`
- `ps -af` is the process we just run to see the processes.


