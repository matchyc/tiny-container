# tiny-container

Build a container from scratch in Golang.

>The content is ... untidy, I'm so sorry for that

**Content**
- [tiny-container](#tiny-container)
  - [Usage](#usage)
  - [PROGRAM 1](#program-1)
  - [Namespace](#namespace)
  - [PROGRAM 2](#program-2)
    - [Check the function of namespce by modifying the hostname66](#check-the-function-of-namespce-by-modifying-the-hostname66)
    - [Change hostname automatically](#change-hostname-automatically)
  - [Program 3](#program-3)
  - [Chroot](#chroot)
    - [Chroot](#chroot-1)
    - [Question](#question)
    - [Solution](#solution)
    - [Show](#show)
  - [Isolate theProcess](#isolate-theprocess)
    - [Pid started with number 1](#pid-started-with-number-1)
    - [Ps Question](#ps-question)
    - [Solution](#solution-1)
  - [Isolates Mount](#isolates-mount)
  - [Cgroups](#cgroups)
      - [Processes number limitation test](#processes-number-limitation-test)

I strongly recommand you to listen the GOTO 2018 conference, the part about container presented by Lize Rice. Then combine this doc, you feel some convenient. Because I explained something ambiguous in the presentation.

## Usage

requirements:
- Linux kernel version 4.15 or higher
- Golang 1.16.6 or higher
- A positive progressive heart

```
git clone
cd tiny-container
go run main.go run bash
```
Then do what ever you want inside the tiny container.

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

> 在这里其实我们只是让一个程序运行了我们所指定的子程序。

## Namespace

Now here we should see something in "C".

I mean the kernel of Linux mostly finished in C, so we have to know what happened when we `clone`.

```C
int clone(int (*fn)(void *), void *child_stack, int flags, void *arg);
```

**Return value**:
On success, the thread ID of the child process is returned in the caller's thread of execution. On failure, -1 is returned in the caller's context, no child process will be created, and errno will be set appropriately.

其中，fn 是函数指针，指向程序的指针， 可理解为。`child_stack`就是为子进程分配系统堆栈空间(在 Linux 在系统堆栈空间是 2 页面，1 个页面 4K，在低地址放入了值，这个值就是进程控制块 `task_struct` 的值)。

When we use the `clone()` warpper function, the child dddprocess is created with the `clone()` function, and it(the child process) commences execution by calling the function pointed to by the argument `fn`.

when fn(arg) returns, the child process terminates. (or terminate explicityly by calling exit or receiving a fatal signal.)

child_stack(stack) arugument specifies the location of the stack used by the child process.

Child and the calling process may share memory, it is not possible for they two to execute in the **same** stack. So the calling process must set up memory space for the child stack and pass a pointer to this to clone()(That's argument `stack`).

> Clone() does not provide a means because the caller can inform the kernel of the size of the stack area.

**clone3()**, providing a **superset** of the functionality of the older clone() interface. But the most important arguments are the same as `clone()`, like flags, parent_tid, stack, child_tid...

> task_struct is very important, we can see most of attributes are showed in task_struct.

**flags** 是用来标记，本次 clone 产生的子进程要从父进程继承哪些资源，其中 arg 是传给 fn 子进程的参数。

> explainations from "Linux manual page"

**Namespace**, we have learned namespace in cpp. A namespace wraps a set of system resource in an "space"(abstrctly), which makes it is visible for the processes `within` the namespace(different namespace has different isolated aspects). Changes to the global resource are visible to other processes that are members of the namespace, but invisible to others.

There are several types of namespace on Linux.

| Namespace | Flag            | Isolates resources                   |
| --------- | --------------- | ------------------------------------ |
| Cgroup    | CLONE_NEWCGROUP | Cgroup root directory                |
| IPC       | CLONE_NEWIPC    | System V IPC, POSIX message queues   |
| Network   | CLONE_NEWNET    | Network devices, stacks, ports, etc. |
| Mount     | CLONE_NEWNS     | Mount points                         |
| PID       | CLONE_NEWPID    | Process IDs                          |
| User      | CLONE_NEWUSER   | User and group IDs                   |
| UTS       | CLONE_NEWUSTS   | Hostname and NIS domain name         |

And the `clone()` _system calls_ is used to creates a new process, if the `flags` argument specifies some CLONE_NEW\* flags, the new namespaces are created for each flag, the child process is made a member of those namespaces.

About namespace, there is also some system calls like `setns` to join an existing namespace, `unshare` to move the calling process to a new namespace, `ioctl` is used to discover information about namespaces.

We need to understand `flags`

If the following flag mask is set, how does it means:

- CLONE_PARENT:

  - if `CLONE_PARENT` is set, then the parent of the new child will be the same as that of the calling process. which means, the child process and the calling process become **brothers**.
  - if it is not set, then the child's parent is the calling process.

- CLONE_FILES:

  - if `CLONE_FILES` is set, the calling process and the child process share the same **file descriptor table**. Which means the file descriptors created by the child process and the calling process could be saw by these two processes, both close and changes its associated flag operations will **affected the other process**.(if a process sharing a file descriptor table calls `execve`, its file descriptor table is duplicated(unshared))
  - if it is not set, the child process inherits a copy of all file descriptors opened in the calling process. And the subsequent operations of file descriptors do not affect the other process. But, as the child refer to the same open file description so share file offsets and file status flags.

- CLONE_FS

  - If `CLONE_FS` is set, the caller and the child process share the same filesystem information. Including the root of the filesystem, the current working directory(cwd), the umask. So the call to `chroot`, `chdir`, `umask` by the caller or the child process also affects the other process.
  - If it is not set, the child process works on a copy of the filesystem information of the calling process. Calls to `chroot`, `chdir`, `umask` do not affect the other process.

- CLONE_NEWCGROUP
  - If `CLONE_NEWCROUP` is set, `clone()` creates the process in a **new cgroup namespaces**.
  - If it is not set, the process is created in the same cgroup namespaces as the calling process.

> **Cgroups**

- CLONE_NEWNET
  - If `CLONE_NEWNET` is set, `clone()` creates the process in a **new network namespace**.
  - If ~ is not set, the new process is created in the same network namespace.

> **Network namespace** isolates the resource about networking: network devices, IPv4\6 protocol stacks, ip tables, firewall rules...

- CLONE_NEWNS
  - If `CLONE_NEWNS` is set, the child process is created in a new mount namespace.
  - If ~ is not set, the child process is created in the same mount namespaces as the parent.

> **Mount namaspace** isolates the list of mount points. Processes will see the different single-directory hierarchies.

- CLONE_NEWUSER
  - get new
  - same

> **User namespace** isolates the attributes associated with security. e.g. user IDs, group IDs, the root directory, keys, etc. A user may has privileges inside a user namespace, at the same time has unprivileged for operations outside the namespace.

- CLONE_NEWUTS
  - get new
  - same

> **UTS(Unix time sharing System) namespace** isolates of two system identifiers. Hostname and NISdomain name. Changes made to the two attributes are visible to all processes in the same UTS namespace, but are not to processes outside the namespace.

CLONE_THREAD
CLONE_VFORK
CLONE_VM - share memory

vfork
vfork() is a special case of clone(). It is used to create new processes **without** copying the page tables of the parent process. The calling thread **suspended** until the child terminates.
......

---

## PROGRAM 2

### Check the function of namespce by modifying the hostname66

Before we execute the command we indicated, we set Coneflags `CLONE_NEWUTS` for the child process.

```go
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS, //isolates system identifier like hostname...
	}
```

Then let's see what will happen.

we use `go run main.go run bash` to execute the bash.

```s
root@ali-ecs:/home/meng/projects/tiny-container# go run main.go run bash
Runnning comand: [bash]
root@ali-ecs:/home/meng/projects/tiny-container# ps -af
UID          PID    PPID  C STIME TTY          TIME CMD
root      268368  268190  0 13:51 pts/2    00:00:00 su root
root      268369  268368  0 13:51 pts/2    00:00:00 bash
root      268454  268369  4 13:51 pts/2    00:00:00 go run main.go run bash
root      268493  268454  0 13:52 pts/2    00:00:00 /tmp/go-build2189013451
root      268498  268493  0 13:52 pts/2    00:00:00 bash
root      268533  268498  0 13:52 pts/2    00:00:00 ps -af
```

As we can see, we use `ps -af` to see the processes in all terminals, we can see `bash` is the program we just ran.

Then we change `hostname` in this blocked bash process.

```shell
root@ali-ecs:/home/meng/projects/tiny-container# hostname
ali-ecs
root@ali-ecs:/home/meng/projects/tiny-container# hostname newhostname
root@ali-ecs:/home/meng/projects/tiny-container# hostname
newhostname
```

In a new terminal we check whether the `hostname` was changed.

```sh
(base) meng@ali-ecs:~/projects/tiny-container$ hostname
ali-ecs
(base) meng@ali-ecs:~/projects/tiny-container$
```

Nothing happened.

Because we **isolates** the Unix time sharing system by using **CLONE_NEWUTS**.

The first terminal only can see the hostname within it's namespace, and the changes in the isolated resources are not visible for process outside the namespace and **$\color{red}{vice\ sersa}$**.

On VM

```
(base) meng@ali-ecs:~/projects/tiny-container$ sudo hostname invm
[sudo] password for meng:
(base) meng@ali-ecs:~/projects/tiny-container$ hostname
invm
```

In namespace

```
root@ali-ecs:/home/meng/projects/tiny-container# hostname
newhostname
root@ali-ecs:/home/meng/projects/tiny-container#
```

### Change hostname automatically

We execute the main.go in twice.

- First time - `func run()`: we create a child process in a **new** UTS_namespace, and the child process calls itself(main.go) to run again.(use trick `/proc/self/exe` always links to the running executable)
- Second time - `func child()`: the second child receives parameters from the first time, and run the expected command **without** creating a new UTS_namespace.

Then we add `syscall.Sethostname([]byte("tiny-container"))` in `child()` function to set hostname as "tiny-container" automatically.

Let see the result.

```sh
root@ali-ecs:/home/meng/projects/tiny-container# go run main.go run bash
Runnning command: [bash]
Running [bash]
root@tiny-container:/home/meng/projects/tiny-container# hostname
tiny-container
root@tiny-container:/home/meng/projects/tiny-container#
```

## Program 3

We need to give the "tiny container" its own set of files and directories, which means we need to limit its view of filesystems.

Using Alpine & Ubuntu respectively Linux to do these things.

Enter the following commands to get Linux alpine.

```
mkdir alpine
cd alpine
curl -o alpine.tar.gz http://dl-cdn.alpinelinux.org/alpine/v3.10/releases/x86_64/alpine-minirootfs-3.10.0-x86_64.tar.gz
rm alpine.tar.gz
```

So we get these directories.

```
(base) meng@ali-ecs:~/projects/tiny-container/alpine$ ls
bin  etc   lib    mnt  proc  run   srv  tmp  var
dev  home  media  opt  root  sbin  sys  usr
```

## Chroot

Using system call `chroot()` changes the root directory of the calling process.

**What chroot() can do:**

> To change an ingredient in the pathname resolution process and does nothiong else.

**Unsafe**. Unless One could ensure no directory will be moved out from chroot directory.

### Chroot

Using `chroot()` and `chdir()` are enough to make tiny-container having its own view of filesystem.

Normally, we just run the command after chroot()\chdir().

### Question

But, here gonna be something wrong if the system outside the container has different file path of the executable.

Such as executable file `ls` is in `/usr/bin` in Ubuntu, but in `/bin` in Alpine.

If we get the `Command` struct in golang before we call `chroot()` and `chdir()`, it will triger a panic like this

```
panic: running: fork/exec /usr/bin/ls: no such file or directory
```

It means our process can't find executable in the **new** root directory. **Because** when this line is executed,

```go
cmd := exec.Command(os.Args[2], os.Args[3:]...)
```

An attribute named `Path` in Command struct will be set as the return value.(exec.Command will return the path and args) After we calling chroot and chdir, the atrribute `Path` in Command struct is not changed. So the program may can't execute the specific executable via `Path` property.

### Solution

There are two ways to solve the problem.

1. Call function `LookPath(file string)` after chroot and chdir.
   `LookPaht(file string) (string, error)` returns the new path of indicated executable and error messages.
   Then we set cmd.Path as the return value.

```go
	//Ubuntu's executables file paths are different from alpine
	//So after chroot and chdir, we need to use LookPath to
	//get the program path and make cmd.Path equals progPath
	progPath, err := exec.LookPath(os.Args[2])
	cmd.Path = progPath
```

2. Just create the variable `cmd` after chroot() and chdir().

### Show

Now the container only can run the executables in its own directory and can't access the file outside the chroot normally.

```
root@ali-ecs:/home/meng/projects/tiny-container# go run main.go run sh
Runnning command: [sh]
Running [sh]
/ # ls
bin    etc    lib    mnt    proc   run    srv    tmp    var
dev    home   media  opt    root   sbin   sys    usr
/ # pwd
/
/ # cd ../../
/ # cd /
/ # cat ../main.go
cat: can't open '../main.go': No such file or directory
/ # ls ../../
bin    etc    lib    mnt    proc   run    srv    tmp    var
dev    home   media  opt    root   sbin   sys    usr
/ #
```

## Isolate theProcess

Logically, we shouldn't see the processes inside the container. But here... we can see the process inside the container from **outside**.
see that...

```
root@ali-ecs:/home/meng/projects/tiny-container# go run main.go run sh
Runnning command: [sh]
Running [sh]
/ # sleep 1000
```

**Another terminal**

```
(base) meng@ali-ecs:~/projects/tiny-container$ ps -C sleep
    PID TTY          TIME CMD
 286698 ?        00:00:00 sleep
 286763 pts/3    00:00:00 sleep
```

so that's the **Sleeping** process with `PID` **286763**.

Let's see `/proc` files about **286763**.

```sh
root@ali-ecs:/home/meng/projects/tiny-container# ls /proc/286763/
arch_status      fd          ns             setgroups
attr             fdinfo      numa_maps      smaps
autogroup        gid_map     oom_adj        smaps_rollup
auxv             io          oom_score      stack
cgroup           limits      oom_score_adj  stat
clear_refs       loginuid    pagemap        statm
cmdline          map_files   patch_state    status
comm             maps        personality    syscall
coredump_filter  mem         projid_map     task
cpuset           mountinfo   root           timers
cwd              mounts      sched          timerslack_ns
environ          mountstats  schedstat      uid_map
exe              net         sessionid      wchan
```

We can see the `root` directory, then let's see the details of root directory.

It's a link to /alpine

```
root@ali-ecs:/home/meng/projects/tiny-container# ls -l /proc/286763/root
lrwxrwxrwx 1 root root 0 Aug 12 21:02 /proc/286763/root -> /home/meng/projects/tiny-container/alpine
```

### Pid started with number 1

FLags of `clone`

- CLONE_NEWPID
  - if `CLONE_NEWPID` is set, the created process will be put into a new pid namespace. And has **PID 1**. And this process become the `init` process inside container, it becomes the parent of any child processes that are **orphaned**.(_orphaned, interesting word_)
  - if not, it's created in the same pid namespace as the calling process.

> PID namespaces isolate the process ID number space, meaning that processes in different PID namespaces can have the same PID. PID namespaces allow containers to provide functionality such as suspending/resuming the set of processes in the container and migrating the container to a new host while the processes inside the container maintain the same PIDs.

In `main.go`, we add this flag.

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
	}
```

### Ps Question

What if we use `ps` and `ls /proc` inside?

Nothing happened.

Normally, it should list an entry of the `ps` command we were just running.

```
/ # ls /proc/
/ #
/ # ps
PID   USER     TIME  COMMAND
/ #
```

**Because** `/proc` is a **pseudo** filesystem.

> It's a machanism for the kernel and the user space to share infomation.

In fact, `ps` finds out about running programs by looking in the `/proc` directory.

And at this moment slash proc(`/proc`) inside the tiny-container in the chroot filesystem has nothing in it. And we should **mount** the a directory as a proc pseudo filesystem to let the kernel knows it need to populate that with information about these running processes.

Use command `mount` in tiny-container's shell:

```
/ # mount
mount: no /proc/mounts
/ #
```

So, we have to modify `main.go` to add a mount point for the proc pseudo filesystem.

### Solution

Add the two lines before and after `cmd.Run()`.

```go
err = syscall.Mount("proc", "proc", "proc", 0, "")

syscall.Unmount("/proc", 0)
```

Then use `ps` in tiny container's `shell` again, we can see processes are listed with `PID` 1. And not any other processes outside the container are listed by `ps` command executed inside the tiny-container.

```
/ # ps
root@ali-ecs:/home/meng/projects/tiny-container# go run main.go run sh
Runnning command: [sh] as 294129
Running [sh] as 1
/ # ps
PID   USER     TIME  COMMAND
    1 root      0:00 /proc/self/exe child sh
    5 root      0:00 sh
    6 root      0:00 ps
```

Use command mount to check mounted filesystems.

- inside the tiny-container

```
/ # mount
proc on /proc type proc (rw,relatime)
```

- outside the tiny-container, we can see the following output.

```
(base) meng@ali-ecs:~/projects/tiny-container$ mount | grep proc
proc on /proc type proc (rw,nosuid,nodev,noexec,relatime)
systemd-1 on /proc/sys/fs/binfmt_misc type autofs (rw,relatime,fd=28,pgrp=1,timeout=0,minproto=5,maxproto=5,direct,pipe_ino=13148)
proc on /home/meng/projects/tiny-container/alpine/proc type proc (rw,relatime)
```

## Isolates Mount

Mount namespaces provide isolations for the mount points seen by processes.

Processes in each mount namespace will see the distinct single-directory hierarchies.

Information of mount is provided by `/proc/[pid]/mounts`, /`proc/[pid]/mountinfo`, `/proc/[pid]/mountstats`.

By using link `mechanism`, processes in the same mount namespace will see the same view in these files.

- **CLONE_NEWNS**
  - If it is set, the child is started in a new mount namespace, initialized with a copy of the namespace of the parent.
  - If not, the child lives in the same mount namespace as the calling process.

> Liz Rice: Apparently, CLONE_NEWNS may be the first clone flags about namespace invented by Linux developer and added into kernel. They may think we don't need other namespaces at that time, so they called that `namespace`. But it's actually for `mount`.

**Unshre flags**:CLONE_NEWNS, get a private copy of its namespace.
In `main.go`: add flags.

```go
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
```

After that, we may can't see chroot /proc when we use command `mount` in our host machine.

- Inside tiny container

```
/ # mount
proc on /proc type proc (rw,relatime)
/ #
```

- Outside tiny container

```
(base) meng@ali-ecs:~/projects/tiny-container$ mount | grep procproc on /proc type proc (rw,nosuid,nodev,noexec,relatime)
systemd-1 on /proc/sys/fs/binfmt_misc type autofs (rw,relatime,fd=28,pgrp=1,timeout=0,minproto=5,maxproto=5,direct,pipe_ino=13148)
binfmt_misc on /proc/sys/fs/binfmt_misc type binfmt_misc (rw,nosuid,nodev,noexec,relatime)
(base) meng@ali-ecs:~/projects/tiny-container$
```

## Cgroups

Control gourps in Linux could be used to limit the hardware resource.

Literally, we can see many kinds of resource in directory `/sys/fs/cgroup`.

For an instance, let's see something about memory.

In `/sys/fs/cgroup/memory/docker`, there are many files to control the limitation for **docker**.

Let's see `memory.limit_in_bytes`

```
cat memory.limit_in_bytes
9223372036854771712
```

That's $2^{63} bytes$ namely unlimited.

we can directly rewrite these files to get control, but that not automatically when we start a container.

Back to `main.go`

This time we use **ubuntu** as rootfs(for using bash to test).

In main.go, we define a function `group()` and call it in `child()`. Just to limit the max number of processes inside the `tiny` namespace.

After that we should test it.

- Get into container using `bash`.

```go
root@ali-ecs:/home/meng/projects/tiny-container# go run main.go run bash
Runnning command: [bash] as 312824
Running [bash] as 1
root@tiny-container:/#
```

- In host machine, let's check files about cgroup.

```go
(base) meng@ali-ecs:cd /sys/fs/cgroup/pids/tiny
(base) meng@ali-ecs:/sys/fs/cgroup/pids/tiny$ cat pids.max
20
```

Which means, the maxium number of processes inside the namespace is **20**.

#### Processes number limitation test

In bash, we define a function `a()`

```
root@tiny-container:/# a() { a | a & }; a
```

`a()` call `a()` inside `a()` and pipe the output to `a()` running in backgroud. After the semicolon, we invoke `a()`.

That's definitely a disaster if we run this command in host machine. You even can't use `ps` to kill these processes.

But, we will run it inside the `tiny` container. Let's see what will happen.

```
root@tiny-container:/# a() { a | a & }; a
[1] 12
root@tiny-container:/# bash: fork: retry: Resource temporarily unavailable
bash: fork: retry: Resource temporarily unavailable
bash: fork: retry: Resource temporarily unavailable
bash: fork: retry: Resource temporarily unavailable
bash: fork: retry: Resource temporarily unavailable
......
```

**Luckily**, I'm alive. Ali light application server didn't die.

If you use `Ctrl + c` and `ps`.

You can still do nothing. Because `ps` requires to start a new process, in this case no more pid for `ps`.
```
[1]+  Done                    a | a
root@tiny-container:/# 
root@tiny-container:/# ps
bash: fork: retry: Resource temporarily unavailable
```

**In host machine**, we can see the current process number.
```go
(base) meng@ali-ecs:/sys/fs/cgroup/pids/tiny$ cat pids.current 
20
```

We limit process number in `tiny` namespace successfully.

Tiny container, Done!

>Namespaces: Control what you can see
>Control Group: Control what you can use

Thanks to Liz Rice, an awesome woman.

Mostly, the contents are similar to the demo in GOTO 2018. But luckily, I understand what happened when we use container, how to limit the view and resource using system call provided by Linux kernel.

>To be honest, I even use sevral days to learn how to use Golang. I always use cpp and python before I trying to learn this procedure. :)

**Reference**
>Containers From Scratch • Liz Rice • GOTO 2018
>Golang documents https://pkg.go.dev/
>Linux manual page https://manual.cs50.io/
>GOTO 2018 demo source code https://github.com/lizrice/containers-from-scratch


