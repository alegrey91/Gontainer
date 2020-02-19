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
const version = "0.8.3"
const shell = "/bin/sh"

/*
Struct to maintain cli flags.
*/
type Opts struct {
	mount      string
	uts        bool
	hostname   string
	ipc        bool
	network    bool
	process_id bool
	user_id    bool
	version    bool
	run        bool
	ns         bool
	//	cmd string
}

func main() {

	/*
		Command line flags management
	*/
	opt := new(Opts)
	flag.Usage = func() {
		help()
	}
	flag.StringVar(&opt.mount, "mnt", "", "")
	flag.BoolVar(&opt.uts, "uts", false, "")
	flag.StringVar(&opt.hostname, "hostname", "", "")
	flag.BoolVar(&opt.ipc, "ipc", false, "")
	flag.BoolVar(&opt.network, "net", false, "")
	flag.BoolVar(&opt.process_id, "process_id", false, "")
	flag.BoolVar(&opt.user_id, "uid", false, "")
	flag.BoolVar(&opt.run, "run", false, "")
	flag.BoolVar(&opt.ns, "ns", false, "")
	flag.BoolVar(&opt.version, "v", false, "")
	//	flag.StringVar(&opt.cmd, "c", "/bin/sh", "")
	flag.Parse()

	flag_code := gen_cloneflags(*opt)

	print_version(opt)

	switch os.Args[1] {
	case "-run":
		run(flag_code)
	case "-ns":
		run_with_ns(opt)
	default:
		//panic("Error!")
		fmt.Println("Wrong arguments passed.")
		help()
		os.Exit(1)
	}
}

/*
Print help.
*/
func help() {
	fmt.Println("Usage: ./Gontainer -run -uid [-mnt=/path/rootfs] [-uts [-hostname=new_hostname]] [-ipc] [-net] [-pid]")
	fmt.Println("  -mnt='/path/rootfs'		Enable Mount namespace")
	fmt.Println("  -uts				Enable UTS namespace")
	fmt.Println("  -hostname='new_hostname'	Set a custom hostname into the container")
	fmt.Println("  -ipc				Enable IPC namespace")
	fmt.Println("  -net				Enable Network namespace")
	fmt.Println("  -pid				Enable PID namespace")
	fmt.Println("  -uid				Enable User namespace")
	fmt.Println("  -v				Check " + program_name + " version")
}

func print_version(opt *Opts) {
	if opt.version {
		fmt.Println(program_name + " v" + version)
		os.Exit(0)
	}
}

/*
Function to debug cli flags values.
*/
func opts_debug(opt *Opts) {

	enabled := "\033[1;32menabled\033[0m"
	disabled := "\033[1;31mdisabled\033[0m"

	fmt.Println("[Gontainer config]")
	if opt.mount != "" {
		fmt.Printf("• mount:  \"%v\"\n", opt.mount)
	} else {
		fmt.Println("• mount:  \"\"")
	}

	if opt.uts != false {
		fmt.Println("• uts: ", enabled)
		if opt.hostname != "" {
			fmt.Printf(" ↳ %s\n", opt.hostname)
		}
	} else {
		fmt.Println("• uts: ", disabled)
	}

	if opt.ipc != false {
		fmt.Println("• ipc: ", enabled)
	} else {
		fmt.Println("• ipc: ", disabled)
	}

	if opt.network != false {
		fmt.Println("• network: ", enabled)
	} else {
		fmt.Println("• network: ", disabled)
	}

	if opt.user_id != false {
		fmt.Println("• user_id: ", enabled)
	} else {
		fmt.Println("• user_id: ", disabled)
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
	set_mount(opt)
	set_uts(opt)
	set_ipc(opt)
	set_network(opt)
	set_process_id(opt)
	set_user_id(opt)

	//cmd := exec.Command(container_cmd(opt))
	cmd := exec.Command(shell)
	cmd.Env = []string{"PS1=📦 [$(whoami)@$(hostname)] ~$(pwd) ‣ "}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	unset_process_id(opt)
}

/*
Generate value to enable selected namespaces.
The function parse the 'opt' struct to generate
final int value using binary OR operator.
It copies a bit if it is existing in either operand.
*/
func gen_cloneflags(opt Opts) int {
	flag_code := 0
	//	if opt.mount != "" {
	if _, err := os.Stat(opt.mount); !os.IsNotExist(err) {
		flag_code = flag_code | syscall.CLONE_NEWNS
	}
	if opt.uts != false {
		flag_code = flag_code | syscall.CLONE_NEWUTS
	}
	if opt.ipc != false {
		flag_code = flag_code | syscall.CLONE_NEWIPC
	}
	if opt.network != false {
		flag_code = flag_code | syscall.CLONE_NEWNET
	}
	if opt.process_id != false {
		flag_code = flag_code | syscall.CLONE_NEWPID
	}
	if opt.user_id != false {
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
by cli (with -hostname flag).
*/
func set_mount(opt *Opts) bool {
	if _, err := os.Stat(opt.mount); !os.IsNotExist(err) {
		if err := syscall.Chroot(opt.mount); err != nil {
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
by cli (with -hostname flag), otherwise set the
default hostname of the program.
*/
func set_uts(opt *Opts) bool {
	var hostname string
	if opt.uts != false {
		if opt.hostname != "" {
			hostname = opt.hostname
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
func set_network(opt *Opts) bool {
	if opt.network == false {
		return false
	}
	return true
}

/*
Set PID namespace environment, checking
'opt' struct to retrieve passed arguments.
Specifically, it mount the 'proc' fs.
*/
func set_process_id(opt *Opts) bool {
	// Check if option mount was set, if not, return false
	if opt.mount != "" {
		if opt.process_id != false {
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
		if opt.process_id != false {
			fmt.Println("Error: option -process_id require -mount.")
		}
		return false
	}
	return true
}

/*
Unset process_id namespace environment.
Umount /proc from filesystem.
*/
func unset_process_id(opt *Opts) bool {
	if opt.mount != "" {
		if opt.process_id != false {
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
Specifically, it map your user_id/gid, with the
container user_id/gid.
*/
func set_user_id(opt *Opts) bool {
	if opt.user_id == false {
		return false
	}
	return true
}
