// COPYRIGHT (c) 2019-2021 SILVANO ZAMPARDI, ALL RIGHTS RESERVED.
// The license for these sources can be found in the LICENSE file in the root directory of this source tree.

package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	log "github.com/szampardi/msg"
	"github.com/szampardi/xprint/temple"
)

var (
	l          log.Logger                                                                                    //
	data       = make(map[string]interface{})                                                                //
	dataIndex  []string                                                                                      //
	name                                      = flag.String("n", os.Args[0], "set name for verbose logging") //
	logfmt     log.Format                     = log.Formats[log.PlainFormat]                                 //
	loglvl     log.Lvl                        = log.LNotice                                                  //
	logcolor                                  = flag.Bool("c", false, "colorize output")                     ////
	_templates []struct {
		S      string
		IsFile bool
	}
	showFns               *bool    = flag.Bool("H", false, "print available template functions and exit")                              //
	debug                 *bool    = flag.Bool("D", false, "debug init and template rendering activities")                             //
	output                *os.File                                                                                                     //
	argsfirst             *bool    = flag.Bool("a", false, "output arguments (if any) before stdin (if any), instead of the opposite") //
	showVersion           *bool    = flag.Bool("v", false, "print build version/date and exit")                                        //
	server                *string  = flag.String("s", "", "start a render server on given address")                                    //
	semver, commit, built          = "v0.0.0-dev", "local", "a while ago"                                                              //
)

func unsafeMode() bool {
	envvar, err := strconv.ParseBool(os.Getenv("XPRINT_UNSAFE"))
	if err != nil {
		return false
	}
	return envvar
}

func logFmts() []string {
	var out []string
	for f := range log.Formats {
		if !strings.Contains(f, "rfc") {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

func setFlags() {
	flag.BoolVar(&temple.EnableUnsafeFunctions, "u", unsafeMode(), fmt.Sprintf("allow evaluation of dangerous template functions (%v)", temple.FnMap.UnsafeFuncs()))
	flag.Func(
		"F",
		fmt.Sprintf("logging format (prefix) %v", logFmts()),
		func(value string) error {
			if v, ok := log.Formats[value]; ok {
				logfmt = v
				return nil
			}
			return fmt.Errorf("invalid format [%s] specified", value)
		},
	)
	flag.Func(
		"l",
		"log level",
		func(value string) error {
			i, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			loglvl = log.Lvl(i)
			return log.IsValidLevel(i)
		},
	)
	flag.Func(
		"t",
		`template(s) (string). this flag can be specified more than once.
the last template specified in the commandline will be executed,
the others can be accessed with the "template" Action.
`,
		func(value string) error {
			_templates = append(_templates, struct {
				S      string
				IsFile bool
			}{value, false})
			return nil
		},
	)
	flag.Func(
		"f",
		`template(s) (files). this flag can be specified more than once.
the last template specified in the commandline will be executed,
the others can be accessed with the "template" Action.
`,
		func(value string) error {
			_, err := os.Stat(value)
			if err != nil {
				return err
			}
			_templates = append(_templates, struct {
				S      string
				IsFile bool
			}{value, true})
			return nil
		},
	)
	flag.Func(
		"o",
		"output to (default is stdout for rendered templates/logs, stderr for everything else)",
		func(value string) error {
			switch value {
			case "", "1", "stdout", "/dev/stdout", os.Stdout.Name():
				output = os.Stdout
				return nil
			case "2", "stderr", "/dev/stderr", os.Stderr.Name():
				output = os.Stderr
				return nil
			}
			var err error
			output, err = os.OpenFile(value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0600))
			return err
		},
	)
}

func appendData(args []string) {
	for n, s := range args {
		key := fmt.Sprintf("%s%d", "arg", n)
		data[key] = s
		dataIndex = append(dataIndex, key)
	}
}

func init() {
	var err error
	setFlags()
	for !flag.Parsed() {
		flag.Parse()
	}
	if *showVersion {
		fmt.Fprintf(os.Stderr, "github.com/szampardi/xprint version %s (%s) built %s\n", semver, commit, built)
		os.Exit(0)
	}
	if *showFns {
		_, err = os.Stderr.WriteString(temple.FnMap.HelpText())
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}
	if err := log.IsValidLevel(int(loglvl)); err != nil {
		panic(err)
	}
	l, err = log.New(logfmt.String(), log.Formats[log.DefTimeFmt].String(), loglvl, *logcolor, *name, os.Stdout)
	if err != nil {
		panic(err)
	}
	if output != nil {
		l.SetOutput(output)
	}
	args := flag.Args()
	if *argsfirst {
		appendData(args)
	} else {
		defer appendData(args)
	}
	stdin, err := os.Stdin.Stat()
	if err == nil && (stdin.Mode()&os.ModeCharDevice) == 0 {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			l.Errorf("reading %s: %s", stdin.Name(), err)
		} else {
			data["stdin"] = string(b)
			dataIndex = append(dataIndex, "stdin")
		}
	}
}

func main() {
	if *debug {
		temple.DebugHTTPRequests = true
	}
	temple.StartTracking()
	if *server != "" {
		if output == nil {
			output = os.Stderr
		}
		var err error
		l, err = log.New(log.Formats[log.StdFormat].String(), log.Formats[log.DefTimeFmt].String(), loglvl, *logcolor, *name, output)
		if err != nil {
			panic(err)
		}
		u, err := url.Parse(*server)
		if err != nil {
			panic(err)
		}
		proto := strings.Split(u.Scheme, ":")[0]
		var addr string
		if proto != "unix" {
			addr = net.JoinHostPort(u.Hostname(), u.Port())
		} else {
			addr = u.Hostname()
		}
		lis, err := net.Listen(proto, addr)
		if err != nil {
			panic(err)
		}
		l.Noticef("set up %s listener on %s", proto, lis.Addr().String())
		http.HandleFunc("/render", temple.RenderServer(temple.FnMap))
		http.HandleFunc("/", temple.UIPage)
		panic(http.Serve(lis, nil).Error())
	}
	buf := new(bytes.Buffer)
	if len(_templates) > 0 {
		argTemplates := map[string]string{}
		localTemplates := []string{}
		for n, t := range _templates {
			if !t.IsFile {
				if len(t.S) > 0 {
					argTemplates[fmt.Sprintf("opt%d", n)] = t.S
				}
			} else {
				localTemplates = append(localTemplates, t.S)
			}
		}
		tpl, tplList, err := temple.FnMap.BuildTemplate(temple.EnableUnsafeFunctions, hex.EncodeToString(temple.Random(12)), "", argTemplates, localTemplates...)
		if err != nil {
			panic(err)
		}
		if err := tpl.ExecuteTemplate(buf, tplList[0], data); err != nil {
			fmt.Println(os.Args)
			panic(err)
		}
		if *debug {
			temple.Tracking.Wait()
		}
	} else {
		for _, s := range dataIndex {
			_, err := fmt.Fprintf(buf, "%s", data[s])
			if err != nil {
				panic(err)
			}
		}
	}
	if buf.Len() < 1 {
		os.Exit(0)
	}
	switch loglvl {
	case log.LCrit:
		l.Criticalf("%s", buf.String())
	case log.LErr:
		l.Errorf("%s", buf.String())
	case log.LWarn:
		l.Warningf("%s", buf.String())
	case log.LNotice:
		l.Noticef("%s", buf.String())
	case log.LInfo:
		l.Infof("%s", buf.String())
	case log.LDebug:
		l.Debugf("%s", buf.String())
	}
}
