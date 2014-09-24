package main

import "os"
import "log"
import "fmt"
import "flag"
import "os/exec"
import "syscall"
import "net/url"
import "io/ioutil"

func main() {
	flag.Parse()
	dh := os.Getenv("DOCKER_HOST")
	log.Printf("dh = %s", dh)

	env := os.Environ()

	if dh != "" {
		dhurl, err := url.Parse(dh)
		if err == nil && dhurl.Scheme == "ssh" {
			lport := 2375
			rport := 2375
			user := ""
			host := "xos"

			dest := host
			if user != "" {
				dest = fmt.Sprintf("%s@%s", user, host)
			}

			tmp, err := ioutil.TempFile("", "sdocker_")
			if err != nil {
				log.Fatal(err.Error())
			}
			start := exec.Command("ssh", "-M", "-S", tmp.Name(),
				"-fnNT", "-L", fmt.Sprintf("%d:localhost:%d", lport, rport), dest)
			log.Printf("Tunnel start command is %#v", start)

			defer func() {
				// Close ssh tunnel
				stop := exec.Command("ssh", "-S", tmp.Name(),
					"-O", "exit", dest)
				log.Printf("Tunnel stop command is %#v", stop)
				// Remove temporary file
				os.Remove(tmp.Name())
			}()
			os.Setenv("DOCKER_HOST", "tcp://xos:2375")
			env = os.Environ()
		}
	}
	cmd := exec.Command("docker", flag.Args()...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			os.Exit(waitStatus.ExitStatus())
		}
	}
	waitStatus := cmd.ProcessState.Sys().(syscall.WaitStatus)
	os.Exit(waitStatus.ExitStatus())
}
