// COPYRIGHT (c) 2019-2021 SILVANO ZAMPARDI, ALL RIGHTS RESERVED.
// The license for these sources can be found in the LICENSE file in the root directory of this source tree.

package temple

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
)

type (
	jreq struct {
		Template  string            `json:"template,omitempty"`
		Templates map[string]string `json:"templates,omitempty"`
		Data      interface{}       `json:"data,omitempty"`
	}
	jresp struct {
		Status  int         `json:"status"`
		Results interface{} `json:"results,omitempty"`
		Error   string      `json:"error,omitempty"`
	}
)

func RenderServer(fnMap templeFnMap) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		L.Noticef("new request ( %s %s ) from %s", r.Method, r.URL, r.RemoteAddr)
		if DebugHTTPRequests {
			b, err := httputil.DumpRequest(r, true)
			if err != nil {
				L.Errorf("request ( %s %s ) from %s: error dumping request: %s", r.Method, r.URL, r.RemoteAddr, err)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			L.Debugf("request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, string(b))
		}
		if r.Method != http.MethodPost {
			L.Errorf("rejected request ( %s %s ) from %s: bad method", r.Method, r.URL, r.RemoteAddr)
			bye(w, r)
			return
		}
		post := jreq{
			//		Template:  `hello {{.client}}, it's {{timestamp}}`,
			//		Data:      struct{ client string }{r.RemoteAddr},
			Templates: make(map[string]string),
		}
		multipart := strings.Contains(r.Header.Get("content-type"), "multipart")
		var err error
		if multipart {
			mr, err := r.MultipartReader()
			if err != nil {
				L.Errorf("request ( %s %s ) from %s: error in request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				switch pname := part.FormName(); pname {
				case "template":
					buf := new(bytes.Buffer)
					_, err = io.Copy(buf, part)
					if err != nil {
						L.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					post.Template = buf.String()
				case "data":
					buf := new(bytes.Buffer)
					_, err = io.Copy(buf, part)
					if err != nil {
						L.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					d := buf.Bytes()
					err = json.Unmarshal(d, &post.Data)
					if err != nil {
						post.Data = string(d)
					}
				case "templates":
					buf := new(bytes.Buffer)
					_, err = io.Copy(buf, part)
					if err != nil {
						L.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if buf.Len() > 0 {
						post.Templates[part.FileName()] = buf.String()
					}

				}
			}
		} else {
			if err = json.NewDecoder(r.Body).Decode(&post); err != nil {
				L.Warningf("error processing request ( %s %s ) from %s: json.Decode: %s", r.Method, r.URL, r.RemoteAddr, err)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(jresp{
					Status: http.StatusBadRequest,
					Error:  err.Error(),
				})
				return
			}
		}
		tpl, err := fnMap.BuildTemplate(EnableUnsafeFunctions, "post", post.Template, post.Templates)
		if err != nil {
			L.Errorf("request ( %s %s ) from %s: error building template.Template: %s", r.Method, r.URL, r.RemoteAddr, err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(jresp{
				Status: http.StatusBadRequest,
				Error:  err.Error(),
			})
			return
		}
		buf := new(bytes.Buffer)
		ctypeBuf := bytes.NewBuffer(make([]byte, 512))
		if err := tpl.Execute(io.MultiWriter(buf, ctypeBuf), post.Data); err != nil {
			L.Warningf("error processing request ( %s %s ) from %s: tpL.Execute: %s", r.Method, r.URL, r.RemoteAddr, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(jresp{
				Status: http.StatusInternalServerError,
				Error:  err.Error(),
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		if (buf.Len() < (1 << 20)) && multipart {
			_, err = w.Write([]byte(fmt.Sprintf("%s\n%s", htmlHead, htmlArticle(buf.String()))))
		} else {
			ctype := http.DetectContentType(ctypeBuf.Bytes())
			w.Header().Set("Content-Disposition", "attachment; filename=rendered.out")
			w.Header().Set("Content-Type", ctype)
			_, err = io.Copy(w, buf)
		}
		if err != nil {
			L.Errorf("error sending response to request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, err)
		} else {
			L.Infof("processed request ( %s %s ) from %s", r.Method, r.URL, r.RemoteAddr)
		}
	}
}

func UIPage(w http.ResponseWriter, r *http.Request) {
	L.Noticef("new request ( %s %s ) from %s", r.Method, r.URL, r.RemoteAddr)
	if DebugHTTPRequests {
		b, _ := httputil.DumpRequest(r, true)
		L.Debugf("request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, string(b))
	}
	if r.Method != http.MethodGet || r.URL.Path != "/" {
		L.Errorf("rejected request ( %s %s ) from %s: bad method or path", r.Method, r.URL, r.RemoteAddr)
		bye(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(w, "%s\n%s", htmlHead, htmlForm)
	if err != nil {
		L.Errorf("error writing response to request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, err)
	}
}

const (
	htmlHead = `
<!DOCTYPE html>
<meta name="viewport" charset="utf-8" content="width=device-width, initial-scale=1">
<meta http-equiv="Cache-Control" content="no-cache, no-store, must-revalidate" />
<meta http-equiv="Pragma" content="no-cache" />
<meta http-equiv="Expires" content="0" />
<style type="text/css">
Body {
	background-color:darkgray;
	display:flex;
	font-family: Calibri, Helvetica, sans-serif;
}
.container {
	margin:auto;
	padding: 12px 20px;
    position: absolute;
    top: 50%;
    left: 50%;
	background: #eee;
    -moz-transform: translateX(-50%) translateY(-50%);
    -webkit-transform: translateX(-50%) translateY(-50%);
    transform: translateX(-50%) translateY(-50%);
	border-radius: 3px;
	border: 5px solid;
	box-shadow: 0 1px 2px rgba(0, 0, 0, .1);
	display: inline-block;
	overflow:auto;
}
textarea {
	display: inline-block;
	min-width: 30em;
	min-height: 20em;
	overflow: auto;
	resize: both;
}
#formItem label {
    display: inline-block;
	margin:auto;
	position:absolute;
	padding: 1px 1px;
}
p { white-space: pre-line; }
button {
	margin-left :5px;
	margin-top :5px;
}
code {
    white-space: pre-wrap;
    overflow-wrap: break-word;
	overflow: auto;
}
</style>
<title>xinput render</title>
`
	htmlForm = `
<body>
<div class="container">
<form method="post" action="render" enctype="multipart/form-data">
    <p>
		<label for="text">TEMPLATE</label>
		<textarea class="text" name="template" id="template">hello {{.client}}, it's {{timestamp}}

{{fns}}</textarea>
	</p>
	<p>
		<label for="text">DATA</label>
		<textarea class="text" name="data" id="data">{"client": "You"}</textarea>
	</p>
	<p>
	<div class="col-md-offset-2 col-md-10 btn-group">
		<input type="file" id="templates" name="templates" accept="text/*" multiple/>


		<input type="submit" class="submit" value="Submit" />


		<input type="reset" value="Reset" class="btn btn-danger pull-right"/>
	</div>
	</p>
</form>
</div>
</body>
</html>
`
)

func htmlArticle(text string) string {
	return fmt.Sprintf(
		"\n%s%s%s\n%s",
		`<div class="container">
<article class="all-browsers">
<pre style="max-height: 50em; overflow: scroll;"><code class="codeblock">`,
		text,
		`</code></pre>
</article>`,
		`</div>
</body>
</html>`,
	)
}

func bye(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://www.youtube.com/watch?v=dQw4w9WgXcQ?autoplay=1", http.StatusPermanentRedirect)
}
