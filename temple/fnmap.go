// COPYRIGHT (c) 2019-2021 SILVANO ZAMPARDI, ALL RIGHTS RESERVED.
// The license for these sources can be found in the LICENSE file in the root directory of this source tree.

package temple

import (
	"encoding/json"
	htmlTpl "html/template"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	textTpl "text/template"
	"time"
)

type (
	fn struct {
		_fn         interface{} `json:"-"`
		Description string      `json:"description"`
		Signature   string      `json:"function"`
		Unsafe      bool        `json:"unsafe"`
	}
	templeFnMap map[string]fn
)

// Add a function to the list of available ones (use before FuncMap())
func (t templeFnMap) Fn(name, description string, funct interface{}, unsafe bool) {
	t[name] = fn{
		funct,
		description,
		reflect.TypeOf(funct).String(),
		unsafe,
	}
}

func (t templeFnMap) BuildFuncMap(addUnsafeFuncs bool) textTpl.FuncMap {
	m := make(textTpl.FuncMap)
	for name, info := range t {
		if !info.Unsafe || addUnsafeFuncs {
			m[name] = info._fn
		}
	}
	return m
}

func (t templeFnMap) BuildHTMLFuncMap(addUnsafeFuncs bool) htmlTpl.FuncMap {
	m := make(htmlTpl.FuncMap)
	for name, info := range t {
		if !info.Unsafe || addUnsafeFuncs {
			m[name] = info._fn
		}
	}
	return m
}

func (t templeFnMap) UnsafeFuncs() []string {
	out := []string{}
	for name, info := range t {
		if info.Unsafe {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func (t templeFnMap) HelpText() string {
	b, _ := json.MarshalIndent(t, "", "  ")
	return string(b)
}

/* src/text/template/funcs.go
func builtins() FuncMap {
	return FuncMap{
		"and":      and,
		"call":     call,
		"html":     HTMLEscaper,
		"index":    index,
		"slice":    slice,
		"js":       JSEscaper,
		"len":      length,
		"not":      not,
		"or":       or,
		"print":    fmt.Sprint,
		"printf":   fmt.Sprintf,
		"println":  fmt.Sprintln,
		"urlquery": URLQueryEscaper,

		// Comparisons
		"eq": eq, // ==
		"ge": ge, // >=
		"gt": gt, // >
		"le": le, // <=
		"lt": lt, // <
		"ne": ne, // !=
	}
}
*/

var (
	defaultFnMapHelpText string
	FnMap                = templeFnMap{
		"add": {
			add,
			"add value $2 to map or slice $1, map needs $3 for value's key in map",
			reflect.TypeOf(add).String(),
			false,
		},
		"b64dec": {
			b64dec,
			"base64 decode",
			reflect.TypeOf(b64dec).String(),
			false,
		},
		"b64enc": {
			b64enc,
			"base64 encode",
			reflect.TypeOf(b64enc).String(),
			false,
		},
		"cmd": {
			cmd,
			"execute a command on local host",
			reflect.TypeOf(cmd).String(),
			true,
		},
		"decrypt": {
			decrypt,
			"decrypt data with AES_GCM: $1 ctxt, $2 base64 key, $3 AAD",
			reflect.TypeOf(decrypt).String(),
			false,
		},
		"duration": {
			time.ParseDuration,
			"time.ParseDuration",
			reflect.TypeOf(time.ParseDuration).String(),
			false,
		},
		"encrypt": {
			encrypt,
			"encrypt data with AES_GCM: $1 ptxt, $2 base64 key, $3 AAD",
			reflect.TypeOf(encrypt).String(),
			false,
		},
		"env": {
			env,
			"get environment vars, optionally use a placeholder value $2",
			reflect.TypeOf(env).String(),
			true,
		},
		"fns": {
			fns,
			"get list of available functions",
			reflect.TypeOf(fns).String(),
			false,
		},
		"fromgob": {
			fromgob,
			"gob decode",
			reflect.TypeOf(fromgob).String(),
			false,
		},
		"fromjson": {
			fromjson,
			"json decode",
			reflect.TypeOf(fromjson).String(),
			false,
		},
		"fromyaml": {
			fromyaml,
			"yaml decode",
			reflect.TypeOf(fromyaml).String(),
			false,
		},
		"gunzip": {
			_gunzip,
			"extract GZIP compressed data",
			reflect.TypeOf(_gunzip).String(),
			false,
		},
		"gzip": {
			_gzip,
			"compress with GZIP",
			reflect.TypeOf(_gzip).String(),
			false,
		},
		"hexdec": {
			hexdec,
			"hex decode",
			reflect.TypeOf(hexdec).String(),
			false,
		},
		"hexenc": {
			hexenc,
			"hex encode",
			reflect.TypeOf(hexenc).String(),
			false,
		},
		"http": {
			_http,
			"HEAD|GET|POST, url, body(raw), headers",
			reflect.TypeOf(_http).String(),
			true,
		},
		"is": {
			is,
			"check if $1 is all |upper(case), |lower(case), |int, |float, |float32, |bool or ==$2",
			reflect.TypeOf(is).String(),
			false,
		},
		"join": {
			strings.Join,
			"strings.Join",
			reflect.TypeOf(strings.Join).String(),
			false,
		},
		"lower": {
			strings.ToLower,
			"strings.ToLower",
			reflect.TypeOf(strings.ToLower).String(),
			false,
		},
		"math": {
			math,
			"math operations (+, -, *, /, %, max, min)",
			reflect.TypeOf(math).String(),
			false,
		},
		"pathbase": {
			filepath.Base,
			"filepath.Base",
			reflect.TypeOf(filepath.Base).String(),
			false,
		},
		"pathext": {
			filepath.Ext,
			"filepath.Ext",
			reflect.TypeOf(filepath.Ext).String(),
			false,
		},
		"random": {
			Random,
			"generate a $1 sized []byte filled with bytes from crypto.Rand",
			reflect.TypeOf(Random).String(),
			false,
		},
		"rawfile": {
			rawfile,
			"read raw bytes from a file",
			reflect.TypeOf(rawfile).String(),
			true,
		},
		"split": {
			strings.Split,
			"strings.Split",
			reflect.TypeOf(strings.Split).String(),
			false,
		},
		"string": {
			stringify,
			"convert int/bool to string, retype []byte to string (handle with care)",
			reflect.TypeOf(stringify).String(),
			false,
		},
		"textfile": {
			textfile,
			"read a file as a string",
			reflect.TypeOf(textfile).String(),
			true,
		},
		"timestamp": {
			timestamp,
			"$1 for timezone (default UTC)",
			reflect.TypeOf(timestamp).String(),
			false,
		},
		"togob": {
			togob,
			"gob encode",
			reflect.TypeOf(togob).String(),
			false,
		},
		"tojson": {
			tojson,
			"json encode",
			reflect.TypeOf(tojson).String(),
			false,
		},
		"toyaml": {
			toyaml,
			"yaml encode",
			reflect.TypeOf(toyaml).String(),
			false,
		},
		"trimprefix": {
			strings.TrimPrefix,
			"strings.TrimPrefix",
			reflect.TypeOf(strings.TrimPrefix).String(),
			false,
		},
		"trimsuffix": {
			strings.TrimSuffix,
			"strings.TrimSuffix",
			reflect.TypeOf(strings.TrimSuffix).String(),
			false,
		},
		"upper": {
			strings.ToUpper,
			"strings.ToUpper",
			reflect.TypeOf(strings.ToUpper).String(),
			false,
		},
		"userinput": {
			userinput,
			"get interactive user input (needs a terminal), if $2 bool is provided and true, term.ReadPassword is used. $1 is used as hint",
			reflect.TypeOf(userinput).String(),
			true,
		},
		"writefile": {
			writefile,
			"store data to a file (append if it already exists)",
			reflect.TypeOf(writefile).String(),
			true,
		},
	}
)
