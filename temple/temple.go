package temple

import (
	log "github.com/szampardi/msg"
)

var (
	L                     log.Logger
	DebugHTTPRequests     = false
	EnableUnsafeFunctions = false
)

func init() {
	defaultFnMapHelpText = FnMap.HelpText()
}
