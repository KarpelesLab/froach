package froach

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/KarpelesLab/cloudinfo"
	"github.com/KarpelesLab/goupd"
	"github.com/KarpelesLab/runutil"
)

func Monitor() {
	time.Sleep(5 * time.Second)
	cockroachCheck()

	// initialize ticker only after running once since first run can take longer (cockroach download, etc)
	t := time.NewTicker(time.Minute)

	for _ = range t.C {
		cockroachCheck()
	}
}

func cockroachCheck() {
	defer func() {
		if e := recover(); e != nil {
			slog.Error(fmt.Sprintf("cockroach check panic: %s", e), "event", "froach:check:panic")
		}
	}()

	if err := check(); err != nil {
		slog.Error(fmt.Sprintf("cockroach run error: %s", err), "event", "froach:check:error")
	}
}

func check() error {
	// first, let's check if cockroach is already running
	pids := runutil.PidOf("cockroach")
	if len(pids) > 0 {
		// already got a cockroach process out there
		return nil
	}

	exe, err := Exe()
	if err != nil {
		return err
	}

	// let's get a list of peers
	peers := getAddrs()

	// make cmdline
	cmdline := makeCmdline(peers)

	// prepare command
	c := exec.Command(exe, cmdline...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	slog.Debug(fmt.Sprintf("[pgdb] about to launch: %s", c), "event", "mercury:pgdb:run")

	err = c.Start()
	if err != nil {
		return err
	}

	return nil
}

func makeCmdline(peers []string) []string {
	// check for goupd flags
	if goupd.MODE == "DEV" {
		return []string{
			"start-single-node",
			"--insecure",
			"--store=type=mem,size=50%",
			"--listen-addr=localhost:36257",
			"--sql-addr=localhost:26257",
			"--accept-sql-without-tls",
			"--http-addr",
			"localhost:28081",
		}
	}

	res := []string{
		"start",
		"--store=" + filepath.Join(cachePath(), "db"),
		"--listen-addr=:36257",
		"--sql-addr=localhost:26257",
		"--accept-sql-without-tls",
		"--cache=.25",
		"--certs-dir=" + basePath(), // cert dir
		"--cluster-name",
		"phplatform", // cluster name
		"--http-addr",
		"localhost:28080",
		//"--locality=cloud=gce,region=us-west1,zone=us-west-1b",
		"--unencrypted-localhost-http",
	}

	info, _ := cloudinfo.Load()
	if ip, ok := info.PublicIP.GetFirstV4(); ok {
		res = append(res, "--advertise-addr="+ip.String()+":36257")
	}
	res = append(res, cockroachLocalityArgs(info)...)

	var join []string
	for _, peer := range peers {
		join = append(join, peer+":36257")
	}
	if len(join) > 0 {
		res = append(res, "--join="+strings.Join(join, ","))
	}

	return res
}

func cockroachLocalityArgs(info *cloudinfo.Info) []string {
	res := []string{
		"--locality=" + info.Location.String(),
	}

	region := info.Location.Get("region")
	if region != "" {
		if ip, ok := info.PrivateIP.GetFirstV4(); ok {
			res = append(res, "--locality-advertise-addr=region="+region+"@"+ip.String()+":36257")
		}
	}

	return res
}
