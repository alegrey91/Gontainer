package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

/**
Common constants.
*/
const programName = "Gontainer"
const version = "0.9.1"
const shell = "/bin/sh"

/*
Struct to maintain cli flags.
*/
type Opts struct {
	mount     string
	uts       bool
	hostname  string
	ipc       bool
	network   bool
	processID bool
	userID    bool
	version   bool
	run       bool
	ns        bool
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
	flag.BoolVar(&opt.processID, "pid", false, "")
	flag.BoolVar(&opt.userID, "uid", false, "")
	flag.BoolVar(&opt.run, "run", false, "")
	flag.BoolVar(&opt.ns, "ns", false, "")
	flag.BoolVar(&opt.version, "v", false, "")
	//	flag.StringVar(&opt.cmd, "c", "/bin/sh", "")
	flag.Parse()

	flagCode := genCloneFlags(*opt)

	printVersion(opt)

	switch os.Args[1] {
	case "-run":
		run(flagCode)
	case "-ns":
		runWithNS(opt)
	default:
		help()
		fmt.Println()
		log.Fatal("Wrong arguments passed.")
	}
}

/*
Print help.
*/
func help() {
	fmt.Println("Usage: ./Gontainer -run -uid [-mnt=/path/rootfs] [-uts [-hostname=new_hostname]] [-ipc] [-net] [-pid]")
	fmt.Println("  -mnt='/path/rootfs'        Enable Mount namespace")
	fmt.Println("  -uts                       Enable UTS namespace")
	fmt.Println("  -hostname='new_hostname'   Set a custom hostname into the container")
	fmt.Println("  -ipc                       Enable IPC namespace")
	fmt.Println("  -net                       Enable Network namespace")
	fmt.Println("  -pid                       Enable PID namespace")
	fmt.Println("  -uid                       Enable User namespace")
	fmt.Println("  -v                         Check " + programName + " version")
}

func printVersion(opt *Opts) {
	if opt.version {
		fmt.Println(programName + " v" + version)
		os.Exit(0)
	}
}

/*
Function to debug cli flags values.
*/
func optsDebug(opt *Opts) {

	enabled := "\033[1;32menabled\033[0m"
	disabled := "\033[1;31mdisabled\033[0m"

	fmt.Println("[Gontainer config]")

	if opt.mount != "" {
		fmt.Println("• mount: ", enabled)
		fmt.Printf(" ↳ %s\n", opt.mount)
	} else {
		fmt.Println("• mount: ", disabled)
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

	if opt.processID != false {
		fmt.Println("• pid: ", enabled)
	} else {
		fmt.Println("• pid: ", disabled)
	}

	if opt.userID != false {
		fmt.Println("• userID: ", enabled)
	} else {
		fmt.Println("• userID: ", disabled)
	}

	fmt.Println()
}

/*
Setup selected namespaces.
*/
func run(flagCode int) {

	cmd := exec.Command("/proc/self/exe", append([]string{"-ns"}, os.Args[2:]...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set Namespaces with generated value
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: uintptr(flagCode),
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
func runWithNS(opt *Opts) {

	optsDebug(opt)
	/*
		Makes corresponding namespaces actions,
		if flag was set
	*/

	must(setMount(opt))
	must(setUTS(opt))
	must(setIPC(opt))
	must(setNetwork(opt))
	must(setProcessID(opt))
	must(setUserID(opt))

	//cmd := exec.Command(container_cmd(opt))
	cmd := exec.Command(shell)
	cmd.Env = []string{"PS1=📦 [$(whoami)@$(hostname)] ~$(pwd) ‣ "}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	defer must(unsetProcessID(opt))
}

/*
Generate value to enable selected namespaces.
The function parse the 'opt' struct to generate
final int value using binary OR operator.
It copies a bit if it is existing in either operand.
*/
func genCloneFlags(opt Opts) int {
	flagCode := 0
	//	if opt.mount != "" {
	if _, err := os.Stat(opt.mount); !os.IsNotExist(err) {
		flagCode = flagCode | syscall.CLONE_NEWNS
	}
	if opt.uts != false {
		flagCode = flagCode | syscall.CLONE_NEWUTS
	}
	if opt.ipc != false {
		flagCode = flagCode | syscall.CLONE_NEWIPC
	}
	if opt.network != false {
		flagCode = flagCode | syscall.CLONE_NEWNET
	}
	if opt.processID != false {
		flagCode = flagCode | syscall.CLONE_NEWPID
	}
	if opt.userID != false {
		flagCode = flagCode | syscall.CLONE_NEWUSER
	}
	return flagCode
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
func setMount(opt *Opts) (bool, error) {
	if opt.mount != "" {
		if _, err := os.Stat(opt.mount); !os.IsNotExist(err) {
			if err := syscall.Chroot(opt.mount); err != nil {
				return false, errors.New("error setting MNT namespace")
			}
			if err := syscall.Chdir("/"); err != nil {
				return false, errors.New("error changing dir")
			}
		} else {
			return false, errors.New("error setting MNT namespace")
		}
	}
	return true, nil
}

/*
Set UTS namespace environment, checking
'opt' struct to retrieve passed arguments.
Specifically set the provided hostname passed
by cli (with -hostname flag), otherwise set the
default hostname of the program.
*/
func setUTS(opt *Opts) (bool, error) {
	var hostname string
	if opt.uts != false {
		if opt.hostname != "" {
			hostname = opt.hostname
		} else {
			hostname = programName
		}
		if err := syscall.Sethostname([]byte(hostname)); err != nil {
			return false, errors.New("error setting UTS namespace")
		}
	}
	return true, nil
}

/* TODO
Set IPC namespace environment, checking
'opt' struct to retrieve passed arguments.
*/
func setIPC(opt *Opts) (bool, error) {
	return true, nil
}

/* TODO
Set NET namespace environment, checking
'opt' struct to retrieve passed arguments.
*/
func setNetwork(opt *Opts) (bool, error) {
	return true, nil
}

/*
Set PID namespace environment, checking
'opt' struct to retrieve passed arguments.
Specifically, it mount the 'proc' fs.
*/
func setProcessID(opt *Opts) (bool, error) {
	// Check if option mount was set, if not, return false
	if opt.mount != "" {
		if opt.processID != false {
			if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
				return false, errors.New("error setting PID namespace")
			}
		}
	} else {
		if opt.processID != false {
			return false, errors.New("error: option -pid require -mount")
		}
	}
	return true, nil
}

/*
Unset processID namespace environment.
Umount /proc from filesystem.
*/
func unsetProcessID(opt *Opts) (bool, error) {
	if opt.mount != "" {
		if opt.processID != false {
			if err := syscall.Unmount("/proc", 0); err != nil {
				return false, errors.New("error unsetting PID namespace")
			}
		}
	}
	return true, nil
}

/* TODO
Set UID namespace environment, checking
'opt' struct to retrieve passed arguments.
Specifically, it map your userID/gid, with the
container userID/gid.
*/
func setUserID(opt *Opts) (bool, error) {
	return true, nil
}

/*
Function to check namespacese have been set correctly.
*/
func must(res bool, err error) {
	if res != true && err != nil {
		log.Fatal(err)
	}
}
