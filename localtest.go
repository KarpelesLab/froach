package froach

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type testServer struct {
	cmd   *exec.Cmd // allows tracking cmd.Process if needed
	ended bool
}

var (
	testDsn   string
	testErr   error
	testStart sync.Once
)

// LocalTestServer returns a dsn that can be used for local tests, especially suitable for Go unit tests
//
// This will start a local database with in-memory storage that will shutdown at the end of the tests.
// The database will always start in an empty state, and all data written to it will be lost once the
// execution completes.
func LocalTestServer() (string, error) {
	testStart.Do(func() {
		testDsn, testErr = launchLocalTestServer()
	})

	return testDsn, testErr
}

func launchLocalTestServer() (string, error) {
	// first, let's locate cockroach
	p, err := Exe()
	if err != nil {
		// cockroach not found
		return "", err
	}

	// prepare to run it
	// TODO: allow configuration of ports? Use random ports?
	cmd := exec.Command(p, "start-single-node", "--insecure", "--store=type=mem,size=50%", "--listen-addr=localhost:26259", "--sql-addr=localhost:26258", "--http-addr", "localhost:28081")

	cmd.Stdout = os.Stdout
	stderr, err := cmd.StderrPipe()
	if err != nil {
		// unlikely
		return "", err
	}

	pi := &testServer{
		cmd: cmd,
	}

	go pi.readStdErr(stderr)

	err = cmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start cockroach: %w", err)
	}

	go pi.wait()

	dsn := "postgresql://root@localhost:26258/defaultdb?sslmode=disable"

	// let's try to connect
	for i := 0; i < 120; i++ {
		err = pi.attemptConnect(dsn)
		if err == nil {
			// success!
			return dsn, nil
		}
		// make sure process is still running
		if pi.ended {
			return "", errors.New("cockroach db ended before we could connect to it")
		}

		time.Sleep(time.Second / 2)
	}

	return "", fmt.Errorf("failed to connect to server: %w", err)
}

// readStdErr can be run in a separate thread and will log any error happening
// with cockroach that isn't an Info or a Warning
func (pi *testServer) readStdErr(pipe io.ReadCloser) {
	buf := bufio.NewReader(pipe)
	for {
		lin, err := buf.ReadString('\n')
		if err != nil {
			log.Printf("error: %s", err)
			return
		}

		lin = strings.TrimSpace(lin)

		if len(lin) == 0 {
			continue
		}

		switch lin[0] {
		case 'I', 'W':
			// Info or Warn: do nothin
		default:
			log.Printf("[cockroach] %s", lin)
		}
	}
}

// wait will execute cmd.Wait() to ensure stderr is closed in case the process ends
func (pi *testServer) wait() {
	pi.cmd.Wait()
	pi.ended = true
}

// attemptConnect will attempt to connect to a given dsn, reporting any errors
func (pi *testServer) attemptConnect(dsn string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	c, err := pgconn.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer c.Close(context.Background())

	return c.Ping(context.Background())
}
