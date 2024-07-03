package froach

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/KarpelesLab/fleet"
	"github.com/KarpelesLab/runutil"
)

// monitor will check if cockroachdb is launched every 1 min and launch it if needed
func monitor() {
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

	_, domain := fleet.Self().Name()
	clusterName := domain
	if pos := strings.IndexByte(clusterName, '.'); pos > 0 {
		clusterName = clusterName[:pos]
	} else if pos == 0 {
		// domain shouldn't start with a dot
		clusterName = "database"
	}

	// make cmdline
	cmdline := makeCmdline(clusterName, peers)

	// prepare command
	c := exec.Command(exe, cmdline...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	slog.Debug(fmt.Sprintf("[pgdb] about to launch: %s", c), "event", "froach:run")

	err = c.Start()
	if err != nil {
		return err
	}

	return nil
}
