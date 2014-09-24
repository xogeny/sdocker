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
	// Run a nested docker command and return its exit status
	status := run()
	os.Exit(status)
}

func run() int {
	// Used to turn on debugging output
	verbose := false

	if os.Getenv("SDOCKER_DEBUG") != "" {
		verbose = true
	}

	// Parse command line flags (we will forward these to the
	// underlying docker process)
	flag.Parse()

	// Get the existing value for DOCKER_HOST
	dh := os.Getenv("DOCKER_HOST")

	// Check if it is empty
	if dh != "" {
		// ...and if it as URL
		dhurl, err := url.Parse(dh)
		// ...and if uses the ssh scheme (otherwise, we do absolutely nothing)
		if err == nil && dhurl.Scheme == "ssh" {
			// Get hostname information and split it at the ":" if present
			hostname := strings.Split(dhurl.Host, ":")

			// Now extract out the remote host and remote port number
			host := hostname[0]
			rport := "4243" // This is the default if no remote port is specified

			if len(hostname) == 2 {
				host = hostname[0]
				rport = hostname[1]
			} else if len(hostname) > 2 {
				log.Fatalf("Unable to determine hostname and port from '%s'", dhurl.Host)
			}

			// Now assume the (same) default to open here on the localhost
			lport := "4243"
			if dhurl.Path != "" {
				// Unless there is a path component in the DOCKER_HOST.  In that
				// case, use the path as the port number.  Odd choice for a port
				// number, but nothing else seems better.
				lport = dhurl.Path[1:]
			}

			// Finally, see if they specified a user
			user := ""
			if dhurl.User != nil {
				user = dhurl.User.Username()
			}

			// Now, we need to start building args for the call to 'ssh'
			dest := host
			if user != "" {
				dest = fmt.Sprintf("%s@%s", user, host)
			}

			// Create a temporary file to use as the socket control file
			tmp, err := ioutil.TempFile("", "sdocker_")
			if err != nil {
				log.Fatal(err.Error())
			}
			// We don't need the actual file, just the name.
			os.Remove(tmp.Name())

			if verbose {
				log.Printf("Tunnel from localhost:%s to %s:%s", lport, dest, rport)
			}

			// This command starts the socket
			start := exec.Command("ssh", "-M", "-S", tmp.Name(),
				"-fnNT", "-L", fmt.Sprintf("%s:localhost:%s", lport, rport), dest)
			terr := start.Run()

			// If it fails, we exit
			if terr != nil {
				log.Fatalf("Unable to open tunnel: %v", terr)
			}

			// This will clean everything up when we are done
			defer func() {
				// Close ssh tunnel
				stop := exec.Command("ssh", "-S", tmp.Name(),
					"-O", "exit", dest)
				err = stop.Run()
				if err != nil {
					log.Printf("Error closing tunnel: %v", err)
				}

				if verbose {
					log.Printf("Tunnel closed and control socket file removed")
				}
			}()

			// Now formulate the value of DOCKER_HOST to pass to our nested
			// invocation of docker
			ndh := fmt.Sprintf("tcp://localhost:%s", lport)
			if verbose {
				log.Printf("Running nested docker command with DOCKER_HOST='%s'", ndh)
			}

			// ...and assign it to the environment
			os.Setenv("DOCKER_HOST", ndh)
		}
	}

	// Here is where we call the nested docker command
	cmd := exec.Command("docker", flag.Args()...)

	// We completely connect stdin, stdout and stderr error so this
	// command looks just like the nested docker process
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// We need to set the environment in case it has changed
	cmd.Env = os.Environ()

	// Now run the command and extract the exit status one way...
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			return waitStatus.ExitStatus()
		}
	}

	// ...or another.
	waitStatus := cmd.ProcessState.Sys().(syscall.WaitStatus)
	return waitStatus.ExitStatus()
}
