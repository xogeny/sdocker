package main

import "os"
import "log"
import "fmt"
import "flag"
import "os/exec"
import "strings"
import "syscall"
import "net/url"
import "io/ioutil"

func main() {
	status := run()
	os.Exit(status)
}

func run() int {
	// Parse command line flags (we will forward these to the
	// underlying docker process
	flag.Parse()

	// Get the existing value for DOCKER_HOST
	dh := os.Getenv("DOCKER_HOST")

	// Check if it is empty
	if dh != "" {
		// ...and if it as URL
		dhurl, err := url.Parse(dh)
		// ...and if uses the ssh scheme (otherwise, we do absolutely nothing)
		if err == nil && dhurl.Scheme == "ssh" {
			hostname := strings.Split(dhurl.Host, ":")

			host := hostname[0]
			rport := "4243"
			if len(hostname) == 2 {
				host = hostname[0]
				rport = hostname[1]
			} else if len(hostname) > 2 {
				log.Fatalf("Unable to determine hostname and port from '%s'", dhurl.Host)
			}

			lport := "4243"
			if dhurl.Path != "" {
				lport = dhurl.Path[1:]
			}

			user := ""
			if dhurl.User != nil {
				user = dhurl.User.Username()
			}

			dest := host
			if user != "" {
				dest = fmt.Sprintf("%s@%s", user, host)
			}

			tmp, err := ioutil.TempFile("", "sdocker_")
			log.Printf("tmp = %s", tmp)
			if err != nil {
				log.Fatal(err.Error())
			}

			log.Printf("Tunnel from localhost:%s to %s:%s", lport, dest, rport)

			start := exec.Command("ssh", "-M", "-S", tmp.Name(),
				"-fnNT", "-L", fmt.Sprintf("%s:localhost:%s", lport, rport), dest)
			terr := start.Run()
			if terr != nil {
				log.Fatalf("Unable to open tunnel: %v", terr)
			}

			log.Printf("Tunnel started: %v", start)

			defer func() {
				// Close ssh tunnel
				stop := exec.Command("ssh", "-S", tmp.Name(),
					"-O", "exit", dest)
				// This exist with status code 255, but as far as I can tell,
				// that is normal.  So I ignore the error (nothing I could
				// do about it anyway really.
				stop.Run()

				// Remove temporary file
				os.Remove(tmp.Name())
			}()

			os.Setenv("DOCKER_HOST", fmt.Sprintf("tcp://localhost:%s", lport))
		}
	}
	cmd := exec.Command("docker", flag.Args()...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			return waitStatus.ExitStatus()
		}
	}
	waitStatus := cmd.ProcessState.Sys().(syscall.WaitStatus)
	return waitStatus.ExitStatus()
}
