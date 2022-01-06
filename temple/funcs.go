// COPYRIGHT (c) 2019-2021 SILVANO ZAMPARDI, ALL RIGHTS RESERVED.
// The license for these sources can be found in the LICENSE file in the root directory of this source tree.

package temple

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"

	log "github.com/szampardi/msg"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

func timestamp(tz ...string) time.Time {
	if len(tz) > 0 {
		return time.Now().In(time.FixedZone(tz[0], 0))
	}
	return time.Now().UTC()
}

func _http(method, url string, body interface{}, headers map[string]string) (out *http.Response, err error) {
	method = strings.ToUpper(method)
	defer trackUsage("http", true, out, err, method, url, headers, body)
	var bodyr io.Reader
	switch t := body.(type) {
	case string:
		bodyr = bytes.NewBuffer([]byte(t))
	case []byte:
		bodyr = bytes.NewBuffer(t)
	case io.Reader:
		bodyr = t
	default:
		err = fmt.Errorf("invalid argument %T, supported types: io.Reader, string or []byte", t)
	}
	var req *http.Request
	req, err = http.NewRequest(method, url, bodyr)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	out, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func userinput(title string, hidden ...bool) (out string, err error) {
	defer trackUsage("userinput", true, &out, err, title, hidden[:])
	h := false
	if len(hidden) > 0 {
		h = hidden[0]
	}
	if h {
		log.Noticef("(%s) input secret now, followed by newline", title)
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		out = string(b)
	} else {
		log.Noticef("(%s) reading input, CTRL-D to stop", title)
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		out = strings.TrimSpace(string(b))
	}
	return out, err
}

func togob(in interface{}) (out []byte, err error) {
	defer trackUsage("togob", false, &out, err, in)
	buf := new(bytes.Buffer)
	if err = gob.NewEncoder(buf).Encode(in); err != nil {
		return nil, err
	}
	out = buf.Bytes()
	return out, nil
}

func fromgob(in interface{}) (out interface{}, err error) {
	defer trackUsage("fromgob", false, &out, err, in)
	var todo io.Reader
	switch t := in.(type) {
	case string:
		todo = bytes.NewBuffer([]byte(t))
	case []byte:
		todo = bytes.NewBuffer(t)
	case io.Reader:
		todo = t
	}
	if err = gob.NewDecoder(todo).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func tojson(in interface{}) (out string, err error) {
	defer trackUsage("tojson", false, &out, err, in)
	b, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	out = string(b)
	return out, nil
}

func fromjson(in interface{}) (out interface{}, err error) {
	defer trackUsage("fromjson", false, &out, err, in)
	switch t := in.(type) {
	case string:
		if err := json.Unmarshal([]byte(t), &out); err != nil {
			return nil, err
		}
	case []byte:
		if err := json.Unmarshal(t, &out); err != nil {
			return nil, err
		}
	case io.Reader:
		if err = json.NewDecoder(t).Decode(&out); err != nil {
			return nil, err
		}
	case *http.Response:
		if err = json.NewDecoder(t.Body).Decode(&out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func toyaml(in interface{}) (out string, err error) {
	defer trackUsage("toyaml", false, &out, err, in)
	b, err := yaml.Marshal(in)
	if err != nil {
		return "", err
	}
	out = string(b)
	return out, nil
}

func fromyaml(in interface{}) (out interface{}, err error) {
	defer trackUsage("fromyaml", false, &out, err, in)
	switch t := in.(type) {
	case string:
		if err := yaml.Unmarshal([]byte(t), &out); err != nil {
			return nil, err
		}
	case []byte:
		if err := yaml.Unmarshal(t, &out); err != nil {
			return nil, err
		}
	case io.Reader:
		if err = yaml.NewDecoder(t).Decode(&out); err != nil {
			return nil, err
		}
	default:
		err = fmt.Errorf("invalid argument %T, supported types: io.Reader, string or []byte", t)
		return nil, err
	}
	return out, nil
}

func b64enc(in interface{}) (out string, err error) {
	defer trackUsage("b64enc", false, &out, err, in)
	var b []byte
	switch t := in.(type) {
	case string:
		b = make([]byte, base64.StdEncoding.EncodedLen(len(t)))
		base64.StdEncoding.Encode(b, []byte(t))
		out = string(b)
	case []byte:
		b = make([]byte, base64.StdEncoding.EncodedLen(len(t)))
		base64.StdEncoding.Encode(b, t)
		out = string(b)
	case io.Reader:
		buf := new(bytes.Buffer)
		_, err = io.Copy(base64.NewEncoder(base64.RawStdEncoding, buf), t)
		if err != nil {
			return "", err
		}
		out = buf.String()
	default:
		err = fmt.Errorf("invalid argument %T, supported types: io.Reader, string or []byte", t)
		return "", err
	}
	return out, nil
}

func b64dec(in interface{}) (out []byte, err error) {
	defer trackUsage("b64dec", false, &out, err, in)
	var b []byte
	var n int
	switch t := in.(type) {
	case string:
		b = make([]byte, base64.StdEncoding.DecodedLen(len(t)))
		n, err = base64.StdEncoding.Decode(b, []byte(t))
		if err != nil {
			return nil, err
		}
		out = b[:n]
	case []byte:
		b = make([]byte, base64.StdEncoding.DecodedLen(len(t)))
		n, err = base64.StdEncoding.Decode(b, t)
		if err != nil {
			return nil, err
		}
		out = b[:n]
	case io.Reader:
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, base64.NewDecoder(base64.RawStdEncoding, t))
		if err != nil {
			return nil, err
		}
		out = buf.Bytes()
	default:
		err = fmt.Errorf("invalid argument %T, supported types: io.Reader, string or []byte", t)
		return nil, err
	}
	return out, nil
}

func hexenc(in interface{}) (out string, err error) {
	defer trackUsage("hexenc", false, &out, err, in)
	switch t := in.(type) {
	case string:
		out = hex.EncodeToString([]byte(t))
	case []byte:
		out = hex.EncodeToString(t)
	case io.Reader:
		buf := new(bytes.Buffer)
		_, err = io.Copy(hex.NewEncoder(buf), t)
		if err != nil {
			return "", err
		}
		out = buf.String()
	default:
		err = fmt.Errorf("invalid argument %T, supported types: string or []byte", t)
		return "", err
	}
	return out, nil
}

func hexdec(in interface{}) (out []byte, err error) {
	defer trackUsage("hexdec", false, &out, err, in)
	var b []byte
	var n int
	switch t := in.(type) {
	case string:
		b, err = hex.DecodeString(t)
		if err != nil {
			return nil, err
		}
		out = b[:n]
	case []byte:
		b = make([]byte, hex.DecodedLen(len(t)))
		n, err = hex.Decode(b, t)
		if err != nil {
			return nil, err
		}
		out = b[:n]
	case io.Reader:
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, hex.NewDecoder(t))
		if err != nil {
			return nil, err
		}
		out = buf.Bytes()
	default:
		err = fmt.Errorf("invalid argument %T, supported types: string or []byte", t)
		return nil, err
	}
	return out, nil
}

func _gzip(in interface{}) (out []byte, err error) {
	defer trackUsage("gzip", false, &out, err, in)
	var todo io.Reader
	switch t := in.(type) {
	case string:
		todo = bytes.NewBuffer([]byte(t))
	case []byte:
		todo = bytes.NewBuffer(t)
	case io.Reader:
		todo = t
	default:
		err = fmt.Errorf("invalid argument %T, supported types: io.Reader, string or []byte", t)
		return nil, err
	}
	buf := new(bytes.Buffer)
	gzw, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(gzw, todo)
	if err != nil {
		return nil, err
	}
	if err = gzw.Flush(); err != nil {
		return nil, err
	}
	if err = gzw.Close(); err != nil {
		return nil, err
	}
	out = buf.Bytes()
	return out, nil
}

func _gunzip(in interface{}) (out []byte, err error) {
	defer trackUsage("gunzip", false, &out, err, in)
	var todo io.Reader
	switch t := in.(type) {
	case string:
		// try to go on assuming its base64-encoded
		b, err := b64dec(t)
		if err != nil {
			return nil, err
		}
		todo = bytes.NewBuffer(b)
	case []byte:
		todo = bytes.NewBuffer(t)
	case io.Reader:
		todo = t
	default:
		err = fmt.Errorf("invalid argument %T, supported types: io.Reader or []byte", t)
		return nil, err
	}
	gzr, err := gzip.NewReader(todo)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, gzr); err != nil {
		return nil, err
	}
	if err = gzr.Close(); err != nil {
		return nil, err
	}
	out = buf.Bytes()
	return out, nil
}

func rawfile(in string) (out []byte, err error) {
	defer trackUsage("rawfile", true, &out, err, in)
	f, err := os.Open(in)
	if err != nil {
		return nil, err
	}
	out, err = ioutil.ReadAll(f)
	return out, err
}

func textfile(in string) (out string, err error) {
	defer trackUsage("textfile", true, &out, err, in)
	f, err := os.Open(in)
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	out = string(data)
	return out, nil
}

func writefile(in interface{}, fpath string) (out string, err error) {
	defer trackUsage("writefile", true, "", err, in, fpath)
	f, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0600))
	if err != nil {
		return "", err
	}
	var todo io.Reader
	switch t := in.(type) {
	case string:
		todo = bytes.NewBuffer([]byte(t))
	case []byte:
		todo = bytes.NewBuffer(t)
	case io.Reader:
		todo = t
	default:
		err = fmt.Errorf("invalid argument %T, supported types: io.Reader, string or []byte", t)
		return "", err
	}
	_, err = io.Copy(f, todo)
	return "", err
}

func stringify(in interface{}) (out string, err error) {
	defer trackUsage("string", false, &out, err, in)
	switch t := in.(type) {
	case string:
		out = t
	case int:
		out = strconv.Itoa(t)
	case bool:
		out = strconv.FormatBool(t)
	case []byte:
		out = string(t)
	case io.Reader:
		b, err := consumeReader(t)
		if err != nil {
			return "", err
		}
		out = string(b)
	default:
		err = fmt.Errorf("invalid argument %T, supported types: int, bool and []byte", t)
		return "", err
	}
	return out, nil
}

func encrypt(in interface{}, b64key string, aad string) (out []byte, err error) {
	defer trackUsage("encrypt", false, &out, err, in, b64key, aad)
	var ptxt []byte
	switch t := in.(type) {
	case string:
		ptxt = []byte(t)
	case []byte:
		ptxt = t
	case io.Reader:
		ptxt, err = consumeReader(t)
		if err != nil {
			return nil, err
		}
	default:
		err = fmt.Errorf("invalid argument %T, supported types: string or []byte", t)
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(b64key)
	if err != nil {
		return nil, err
	}
	cb, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(cb)
	if err != nil {
		return nil, err
	}
	iv := Random(aead.NonceSize())
	var ctxt []byte
	ctxt = aead.Seal(ctxt, iv, ptxt, []byte(aad))
	out = append(ctxt, iv...)
	return out, nil
}

func decrypt(in interface{}, b64key string, aad string) (out []byte, err error) {
	defer trackUsage("decrypt", false, &out, err, in, b64key, aad)
	var ctxt []byte
	switch t := in.(type) {
	case string: // try to go on assuming its base64-encoded
		ctxt, err = b64dec(t)
		if err != nil {
			return nil, err
		}
	case []byte:
		ctxt = t
	case io.Reader:
		ctxt, err = consumeReader(t)
		if err != nil {
			return nil, err
		}
	}
	key, err := base64.StdEncoding.DecodeString(b64key)
	if err != nil {
		return nil, err
	}
	cb, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(cb)
	if err != nil {
		return nil, err
	}
	ns := aead.NonceSize()
	l := len(ctxt)
	if l < ns {
		return nil, io.ErrUnexpectedEOF
	}
	out, err = aead.Open(out, ctxt[l-ns:], ctxt[:l-ns], []byte(aad))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func is(s string, what string) bool {
	if strings.HasPrefix(what, "|") {
		switch x := strings.TrimPrefix(what, "|"); x {
		case "u", "upper":
			for _, r := range s {
				if unicode.IsLetter(r) && unicode.IsLower(r) {
					return false
				}
			}
		case "l", "lower":
			for _, r := range s {
				if unicode.IsLetter(r) && unicode.IsUpper(r) {
					return false
				}
			}
		case "i", "int":
			if _, err := strconv.Atoi(s); err != nil {
				return false
			}
		case "f", "float":
			if _, err := strconv.ParseFloat(s, 64); err != nil {
				return false
			}
		case "f32", "float32":
			if _, err := strconv.ParseFloat(s, 32); err != nil {
				return false
			}
		case "b", "bool":
			if s != "true" && s != "false" {
				return false
			}
		default:
			return s == what
		}
	} else {
		return s == what
	}
	return true
}

func math(b, a interface{}, x string) (interface{}, error) {
	switch x {
	case "+", "add":
		return add(b, a)
	case "/", "div":
		return divide(b, a)
	case "max":
		return maximum(b, a)
	case "min":
		return minimum(b, a)
	case "%", "mod":
		return modulo(b, a)
	case "x", "mul":
		return multiply(b, a)
	case "-", "sub":
		return subtract(b, a)
	default:
		return nil, fmt.Errorf("unsupported method %s", x)
	}
}

func mapadd(in interface{}, value interface{}, key ...interface{}) (out interface{}, err error) {
	defer trackUsage("add", false, &out, err, in, value, key)
	switch t := in.(type) {
	case map[int]interface{}:
		if len(key) < 1 {
			return nil, fmt.Errorf("must provide a key for value to be added")
		}
		switch tk := key[0].(type) {
		case int:
			t[tk] = value
			out = t
			return t, nil
		default:
			err = fmt.Errorf("key for value to be added must be an int, not %T", tk)
			return nil, err
		}
	case map[string]interface{}:
		if len(key) < 1 {
			return nil, fmt.Errorf("must provide a key for value to be added")
		}
		switch tk := key[0].(type) {
		case string:
			t[tk] = value
			out = t
			return t, nil
		default:
			err = fmt.Errorf("key for value to be added must be a string, not %T", tk)
			return nil, err
		}
	case map[interface{}]interface{}:
		if len(key) < 1 {
			err = fmt.Errorf("must provide a key for value to be added")
			return nil, err
		}
		// should be safe?
		t[key[0]] = value
		out = t
		return t, nil
	case []interface{}:
		t = append(t, value)
		out = t
		return out, nil
	default:
		err = fmt.Errorf("invalid argument %T, supported types: slices and maps", t)
		return nil, err
	}
}

func env(in string, or ...string) (out string) {
	defer trackUsage("env", true, &out, nil, in, or)
	if v, ok := os.LookupEnv(in); ok {
		return v
	}
	if len(or) > 0 {
		return or[0]
	}
	return ""
}

type cmdBuffers struct {
	stdout       *bytes.Buffer
	stderr       *bytes.Buffer
	ProcessState *os.ProcessState
}

func (c *cmdBuffers) Stdout() string {
	return strings.TrimSpace(c.stdout.String())
}

func (c *cmdBuffers) Stderr() string {
	return strings.TrimSpace(c.stderr.String())
}

func cmd(prog string, args ...string) (out *cmdBuffers, err error) {
	defer trackUsage("cmd", true, &out, err, prog, args[:])
	x := exec.Command(prog, args...)
	out = &cmdBuffers{
		new(bytes.Buffer),
		new(bytes.Buffer),
		nil,
	}
	x.Stderr = out.stderr
	x.Stdout = out.stdout
	err = x.Run()
	if x.ProcessState != nil {
		out.ProcessState = x.ProcessState
	} else {
		out.ProcessState = &os.ProcessState{}
	}
	return out, err
}

func Random(size int) (out []byte) {
	out = make([]byte, size)
	_, err := io.ReadFull(rand.Reader, out)
	if err != nil {
		panic(err)
	}
	return out[:size]
}

func consumeReader(in interface{}) ([]byte, error) {
	var err error
	buf := new(bytes.Buffer)
	switch t := in.(type) {
	case io.Reader:
		_, err = io.Copy(buf, t)
	case *http.Response:
		_, err = io.Copy(buf, t.Body)
	}
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fns() string {
	return defaultFnMapHelpText
}
