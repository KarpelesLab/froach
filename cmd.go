package froach

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/KarpelesLab/cloudinfo"
	"github.com/KarpelesLab/goupd"
	"github.com/KarpelesLab/runutil"
)

func cockroachMonitor() {
	time.Sleep(5 * time.Second)
	cockroachCheck()

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

	// let's get a list of peers
	peers := getAddrs()

	// make cmdline
	cmdline := makeCmdline(peers)

	// prepare command
	c := exec.Command("/pkg/main/dev-db.cockroach-bin.core/bin/cockroach", cmdline...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	slog.Debug(fmt.Sprintf("[pgdb] about to launch: %s", c), "event", "mercury:pgdb:run")

	err := c.Start()
	if err != nil {
		return err
	}

	return nil
}

// if running locally (dev mode), we can run cockroach using:
// /pkg/main/dev-db.cockroach-bin.core/bin/cockroach start-single-node --insecure --store=type=mem,size=50% --listen-addr=localhost:36257 --sql-addr=localhost:26257 --accept-sql-without-tls --http-addr localhost:28080
//
// If running as a cluster, we first need to generate or fetch certificates
// Then we need the following options
// --store=type=ssd,/www/phplatform/stable/data/db
// --listen-addr=:36257
// --advertise-addr <ip>
// --sql-addr=localhost:26257
// --accept-sql-without-tls
// --attrs ?
// --cache .25
// --certs-dir /path/to/certs (/home/magicaltux/.cockroach-certs)
// --cluster-name <name>
// --http-addr localhost:28080
// --join host:port,host:port
// --locality=cloud=gce,region=us-west1,zone=us-west-1b
// --socket-dir ?
// --unencrypted-localhost-http

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

	// /pkg/main/dev-db.cockroach-bin.core/bin/cockroach start --store=type=ssd,/www/phplatform/stable/data/db --listen-addr=:36257 --advertise-addr 172.31.14.173 --sql-addr=localhost:26257 --accept-sql-without-tls --cache .25 --certs-dir /www/phplatform/stable/data/db-cert --cluster-name phplatform --http-addr localhost:28080 --join 172.31.9.176:36257 --join 172.31.41.42:36257 --join 172.31.18.81:36257 --locality=cloud=aws,region=ap-northeast-1,zone=ap-northeast-1c --unencrypted-localhost-http

	res := []string{
		"start",
		"--store=/www/phplatform/stable/data/db",
		"--listen-addr=:36257",
		"--sql-addr=localhost:26257",
		"--accept-sql-without-tls",
		"--cache=.25",
		"--certs-dir=/www/phplatform/stable/data/db-cert", // cert dir
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
