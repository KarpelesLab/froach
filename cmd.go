package froach

import (
	"path/filepath"
	"strings"

	"github.com/KarpelesLab/cloudinfo"
	"github.com/KarpelesLab/goupd"
)

func makeCmdline(peers []string) []string {
	// check for goupd flags
	if goupd.MODE == "DEV" {
		return []string{
			"start-single-node",
			"--insecure",
			"--store=type=mem,size=50%", // will disappear on stop
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
