package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

/*
Common constants.
*/
const program_name = "Gontainer"
const version = "0.7"
const shell = "/bin/sh"

/*
Struct to maintain cli flags.
*/
type Opts struct {
	mnt string
	uts bool
	hst string
	ipc bool
	net bool
	pid bool
	uid bool
	ver bool
	run bool
	ns  bool
	//	cmd string
}

func main() {

	/*
		Command line flags management
	*/
	opt := new(Opts)
	flag.Usage = func() {
		fmt.Println("Usage: ./Gontainer -run -uid [-mnt=/path/rootfs] [-uts [-hst=new_hostname]] [-ipc] [-net] [-pid]")
		fmt.Println("  -mnt='/path/rootfs'	Enable Mount namespace")
		fmt.Println("  -uts			Enable UTS namespace")
		fmt.Println("  -hst='new_hostname'	Set a custom hostname into the container")
		fmt.Println("  -ipc			Enable IPC namespace")
		fmt.Println("  -net			Enable Network namespace")
		fmt.Println("  -pid			Enable PID namespace")
		fmt.Println("  -uid			Enable User namespace")
	}
	flag.StringVar(&opt.mnt, "mnt", "", "")
	flag.BoolVar(&opt.uts, "uts", false, "")
	flag.StringVar(&opt.hst, "hst", "", "")
	flag.BoolVar(&opt.ipc, "ipc", false, "")
	flag.BoolVar(&opt.net, "net", false, "")
	flag.BoolVar(&opt.pid, "pid", false, "")
	flag.BoolVar(&opt.uid, "uid", false, "")
	flag.BoolVar(&opt.run, "run", false, "")
	flag.BoolVar(&opt.ns, "ns", false, "")
	//	flag.StringVar(&opt.cmd, "c", "/bin/sh", "")
	flag.Parse()

	flag_code := gen_cloneflags(*opt)

	switch os.Args[1] {
	case "-run":
		run(flag_code)
	case "-ns":
		run_with_ns(opt)
	default:
		//panic("Error!")
		fmt.Println("Wrong arguments passed.")
		os.Exit(1)
	}
}

/*
Function to debug cli flags values.
*/
func opts_debug(opt *Opts) {

	enabled := "\033[1;32menabled\033[0m"
	disabled := "\033[1;31mdisabled\033[0m"

	fmt.Println("[Gontainer config]")
	if opt.mnt != "" {
		fmt.Printf("• mnt:  \"%v\"\n", opt.mnt)
	} else {
		fmt.Println("• mnt:  \"\"")
	}

	if opt.uts != false {
		fmt.Println("• uts: ", enabled)
		if opt.hst != "" {
			fmt.Printf(" ↳ %s\n", opt.hst)
		}
	} else {
		fmt.Println("• uts: ", disabled)
	}

	if opt.ipc != false {
		fmt.Println("• ipc: ", enabled)
	} else {
		fmt.Println("• ipc: ", disabled)
	}

	if opt.net != false {
		fmt.Println("• net: ", enabled)
	} else {
		fmt.Println("• net: ", disabled)
	}

	if opt.uid != false {
		fmt.Println("• uid: ", enabled)
	} else {
		fmt.Println("• uid: ", disabled)
	}

	fmt.Println()
}

/*
Setup selected namespaces.
*/
func run(flag_code int) {

	cmd := exec.Command("/proc/self/exe", append([]string{"-ns"}, os.Args[2:]...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set Namespaces with generated value
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: uintptr(flag_code),
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}
	cmd.Run()
}

/*
Start containerized process using selected namespaces.
*/
func run_with_ns(opt *Opts) {

	opts_debug(opt)
	/*
		Makes corresponding namespaces actions,
		if flag was set
	*/
	set_mnt(opt)
	set_uts(opt)
	set_ipc(opt)
	set_net(opt)
	set_pid(opt)
	set_uid(opt)

	//cmd := exec.Command(container_cmd(opt))
	cmd := exec.Command(shell)
	cmd.Env = []string{"PS1=📦 [$(whoami)@$(hostname)] ~$(pwd) ‣ "}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	unset_pid(opt)
}

/*
Generate value to enable selected namespaces.
The function parse the 'opt' struct to generate
final int value using binary OR operator.
It copies a bit if it is existing in either operand.
*/
func gen_cloneflags(opt Opts) int {
	flag_code := 0
	//	if opt.mnt != "" {
	if _, err := os.Stat(opt.mnt); !os.IsNotExist(err) {
		flag_code = flag_code | syscall.CLONE_NEWNS
	}
	if opt.uts != false {
		flag_code = flag_code | syscall.CLONE_NEWUTS
	}
	if opt.ipc != false {
		flag_code = flag_code | syscall.CLONE_NEWIPC
	}
	if opt.net != false {
		flag_code = flag_code | syscall.CLONE_NEWNET
	}
	if opt.pid != false {
		flag_code = flag_code | syscall.CLONE_NEWPID
	}
	if opt.uid != false {
		flag_code = flag_code | syscall.CLONE_NEWUSER
	}
	return flag_code
}

/* TODO
Set command to be executed inside the container instance.
func container_cmd(opt *Opts) string {
	cmd := ""
	if opt.cmd != "" {
		cmd = shell + " -c " + opt.cmd
	} else {
		cmd = shell
	}
	return cmd
}
*/

/*
Set MNT namespace environment, checking
'opt' struct to retrieve passed arguments.
Specifically, it chroot the rootfs passed
by cli (with -hst flag).
*/
func set_mnt(opt *Opts) bool {
	if _, err := os.Stat(opt.mnt); !os.IsNotExist(err) {
		if err := syscall.Chroot(opt.mnt); err != nil {
			fmt.Println("Error setting MNT namespace.")
			os.Exit(1)
		}
		if err := syscall.Chdir("/"); err != nil {
			fmt.Println("Error during change dir.")
			os.Exit(1)
		}
	} else {
		return false
	}
	return true
}

/*
Set UTS namespace environment, checking
'opt' struct to retrieve passed arguments.
Specifically set the provided hostname passed
by cli (with -hst flag), otherwise set the
default hostname of the program.
*/
func set_uts(opt *Opts) bool {
	var hostname string
	if opt.uts != false {
		if opt.hst != "" {
			hostname = opt.hst
		} else {
			hostname = program_name
		}
		if err := syscall.Sethostname([]byte(hostname)); err != nil {
			fmt.Println("Error setting UTS namespace.")
			os.Exit(1)
			//panic(err)
		}
	} else {
		return false
	}
	return true
}

/* TODO
Set IPC namespace environment, checking
'opt' struct to retrieve passed arguments.
*/
func set_ipc(opt *Opts) bool {
	if opt.ipc == false {
		return false
	}
	return true
}

/* TODO
Set NET namespace environment, checking
'opt' struct to retrieve passed arguments.
*/
func set_net(opt *Opts) bool {
	if opt.net == false {
		return false
	}
	return true
}

/*
Set PID namespace environment, checking
'opt' struct to retrieve passed arguments.
Specifically, it mount the 'proc' fs.
*/
func set_pid(opt *Opts) bool {
	// Check if option mount was set, if not, return false
	if opt.mnt != "" {
		if opt.pid != false {
			if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
				fmt.Println("Error setting PID namespace.")
				os.Exit(1)
			}
			//			if err := syscall.Mount("dev", "/dev", "devtmpfs", 0, ""); err != nil {
			//				fmt.Println("Error setting PID namespace.")
			//				os.Exit(1)
			//			}
		}
	} else {
		if opt.pid != false {
			fmt.Println("Error: option -pid require -mnt.")
		}
		return false
	}
	return true
}

/*
Unset pid namespace environment.
Umount /proc from filesystem.
*/
func unset_pid(opt *Opts) bool {
	if opt.mnt != "" {
		if opt.pid != false {
			if err := syscall.Unmount("/proc", 0); err != nil {
				fmt.Println("Error unsetting PID namespace.")
				os.Exit(1)
			}
		}
	} else {
		return false
	}
	return true
}

/* TODO
Set UID namespace environment, checking
'opt' struct to retrieve passed arguments.
Specifically, it map your uid/gid, with the
container uid/gid.
*/
func set_uid(opt *Opts) bool {
	if opt.uid == false {
		return false
	}
	return true
}
