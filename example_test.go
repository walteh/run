package run_test

import (
	"time"

	"github.com/walteh/run"
	"github.com/walteh/run/contrib/preempt"
)

func Example() {
	run.Add(true, preempt.New(100*time.Millisecond))

	run.Run()
}
