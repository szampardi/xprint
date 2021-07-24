// COPYRIGHT (c) 2019-2021 SILVANO ZAMPARDI, ALL RIGHTS RESERVED.
// The license for these sources can be found in the LICENSE file in the root directory of this source tree.

package temple

var (
	DebugHTTPRequests     = false
	EnableUnsafeFunctions = false
)

func init() {
	defaultFnMapHelpText = FnMap.HelpText()
}
