package froach

import (
	"time"

	"github.com/KarpelesLab/fleet"
)

func init() {
	go start()
}

func start() {
	fleet.Self().WaitReady() // this will wait for fleet to start
	time.Sleep(time.Second)  // give a bit of time just in case
}
