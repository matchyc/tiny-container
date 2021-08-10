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

>**Cgroups**

- CLONE_NEWNET
  - If `CLONE_NEWNET` is set, `clone()` creates the process in a **new network namespace**.
  - If ~ is not set, the new process is created in the same network namespace.

>**Network namespace** isolates the resource about networking: network devices, IPv4\6 protocol stacks, ip tables, firewall rules...


- CLONE_NEWNS
  - If `CLONE_NEWNS` is set, the child process is created in a new mount namespace.
  - If ~ is not set, the child process is created in the same mount namespaces as the parent.

>**Mount namaspace** isolates the list of mount points. Processes will see the different single-directory hierarchies.

- CLONE_NEWUSER
  - get new
  - same

>**User namespace** isolates the attributes associated with security. e.g. user IDs, group  IDs, the root directory, keys, etc. A user may has privileges inside a user namespace, at the same time has unprivileged for operations outside the namespace.

- CLONE_NEWUTS
  - get new
  - same

>**UTS namespace** isolates of two system identifiers. Hostname and NISdomain name. Changes made to the two attributes are visible to all processes in the same UTS namespace, but are not to processes outside the namespace.

CLONE_THREAD 
CLONE_VFORK
CLONE_VM - share memory

vfork
  vfork() is a special case of clone(). It is used to create new processes **without** copying the page tables of the parent process. The calling thread **suspended** until the child terminates.
......