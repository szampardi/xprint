// COPYRIGHT (c) 2019-2021 SILVANO ZAMPARDI, ALL RIGHTS RESERVED.
// The license for these sources can be found in the LICENSE file in the root directory of this source tree.

package temple

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/szampardi/msg"
)

type (
	fnTrack struct {
		T      time.Time     `json:"time"`
		F      string        `json:"function"`
		Args   []interface{} `json:"args,omitempty"`
		Output interface{}   `json:"output,omitempty"`
		Err    error         `json:"error,omitempty"`
	}
)

var (
	fnTrackChan       chan (*fnTrack) = make(chan *fnTrack)
	startTracingOnce  sync.Once
	debugAllFunctions                 = false
	Tracking          *sync.WaitGroup = &sync.WaitGroup{}
)

func trackUsage(_fn string, alwaysTrack bool, output interface{}, err error, args ...interface{}) {
	if debugAllFunctions || alwaysTrack {
		Tracking.Add(1)
		fnTrackChan <- &fnTrack{
			T:      time.Now(),
			F:      _fn,
			Args:   args[:],
			Output: fmt.Sprintf("%#v", output),
			Err:    err,
		}
	}
}

func usageDebugger() {
	go func() {
		log.SetOutput(os.Stderr)
		for x := range fnTrackChan {
			j, _ := json.Marshal(x)
			log.Warning(string(j))
			Tracking.Done()
		}
	}()
}

func StartTracking() {
	debugAllFunctions = true
	startTracingOnce.Do(usageDebugger)
}

func StopTracking() {
	debugAllFunctions = false
}
