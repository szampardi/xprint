// COPYRIGHT (c) 2019-2021 SILVANO ZAMPARDI, ALL RIGHTS RESERVED.
// The license for these sources can be found in the LICENSE file in the root directory of this source tree.

package temple

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
)

var (
	DebugHTTPRequests     = false
	EnableUnsafeFunctions = false
)

func init() {
	defaultFnMapHelpText = FnMap.HelpText()
}

func (t templeFnMap) BuildTemplate(_unsafe bool, name, _template string, loadedFiles map[string]string, localFiles ...string) (*template.Template, []string, error) {
	var err error
	var all []string
	tpl := template.New(name).Funcs(t.BuildFuncMap(_unsafe))
	if _template != "" {
		tpl, err = tpl.Parse(_template)
		if err != nil {
			return nil, nil, err
		}
		all = []string{path.Base(name)}
	}
	for _, lft := range localFiles {
		text, err := fload(lft)
		if err != nil {
			return nil, nil, err
		}
		n := path.Base(lft)
		tpl, err = tpl.New(n).Parse(text)
		if err != nil {
			return nil, nil, err
		}
		all = append(all, n)
	}
	for fname, content := range loadedFiles {
		fname = path.Base(fname)
		tpl, err = tpl.New(fname).Parse(content)
		if err != nil {
			return nil, nil, err
		}
		all = append(all, fname)
	}
	if len(all) < 1 {
		return nil, all, fmt.Errorf("no templates found")
	}
	return tpl, all, nil
}

func render(t *template.Template, data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fload(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	i := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := scanner.Text()
		if i == 0 && strings.HasPrefix(t, "#!") {
			continue
		}
		_, err = buf.WriteString(t)
		if err != nil {
			return "", err
		}
	}
	return buf.String(), scanner.Err()
}
