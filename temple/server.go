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
	"strconv"
	"strings"

	log "github.com/szampardi/msg"
)

type (
	jreq struct {
		Template  string            `json:"template,omitempty"`
		Templates map[string]string `json:"templates,omitempty"`
		Data      interface{}       `json:"data,omitempty"`
		Outfile   string            `json:"outfile,omitempty"`
		ForceDL   bool              `json:"forcedl,omitempty"`
	}
	jresp struct {
		Status  int         `json:"status"`
		Results interface{} `json:"results,omitempty"`
		Error   string      `json:"error,omitempty"`
	}
)

func RenderServer(fnMap templeFnMap) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Noticef("new request ( %s %s ) from %s", r.Method, r.URL, r.RemoteAddr)
		if DebugHTTPRequests {
			b, err := httputil.DumpRequest(r, true)
			if err != nil {
				log.Errorf("request ( %s %s ) from %s: error dumping request: %s", r.Method, r.URL, r.RemoteAddr, err)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			log.Debugf("request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, string(b))
		}
		if r.Method != http.MethodPost {
			log.Errorf("rejected request ( %s %s ) from %s: bad method", r.Method, r.URL, r.RemoteAddr)
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
				log.Errorf("request ( %s %s ) from %s: error in request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, err)
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
						log.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					post.Template = buf.String()
				case "data":
					buf := new(bytes.Buffer)
					_, err = io.Copy(buf, part)
					if err != nil {
						log.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
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
						log.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if buf.Len() > 0 {
						post.Templates[part.FileName()] = buf.String()
					}
				case "outfile":
					buf := new(bytes.Buffer)
					_, err = io.Copy(buf, part)
					if err != nil {
						log.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					post.Outfile = buf.String()
				case "forcedl":
					buf := new(bytes.Buffer)
					_, err = io.Copy(buf, part)
					if err != nil {
						log.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					post.ForceDL, err = strconv.ParseBool(buf.String())
					if err != nil {
						log.Errorf("request ( %s %s ) from %s: error reading part %s request.MultipartReader: %s", r.Method, r.URL, r.RemoteAddr, pname, err)
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
				}
			}
		} else {
			if err = json.NewDecoder(r.Body).Decode(&post); err != nil {
				log.Warningf("error processing request ( %s %s ) from %s: json.Decode: %s", r.Method, r.URL, r.RemoteAddr, err)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(jresp{
					Status: http.StatusBadRequest,
					Error:  err.Error(),
				})
				return
			}
		}
		tpl, _, err := fnMap.BuildTemplate(EnableUnsafeFunctions, "post", post.Template, post.Templates)
		if err != nil {
			log.Errorf("request ( %s %s ) from %s: error building template.Template: %s", r.Method, r.URL, r.RemoteAddr, err)
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
			log.Warningf("error processing request ( %s %s ) from %s: tplog.Execute: %s", r.Method, r.URL, r.RemoteAddr, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(jresp{
				Status: http.StatusInternalServerError,
				Error:  err.Error(),
			})
			return
		}
		if post.Outfile == "" {
			post.Outfile = "rendered.out"
		}
		w.Header().Set("Expires", "0")
		w.Header().Set("Content-Control", "private, no-transform, no-store, must-revalidate")
		if (buf.Len() < (1 << 20)) && !post.ForceDL && multipart {
			tpl, _, err := FnMap.BuildHTMLTemplate(false, "rendered", renderedPage, nil)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Errorf("error building template for response to request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, err)
				return
			}
			rbuf := new(bytes.Buffer)
			err = tpl.ExecuteTemplate(rbuf, "rendered", struct{ Output string }{buf.String()})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Errorf("error rendering template for response to request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, err)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(rbuf.Bytes())
			//_, err = w.Write([]byte(fmt.Sprintf("%s\n%s", htmlHead, htmlArticle(buf.String()))))
		} else {
			w.Header().Set("Content-Type", http.DetectContentType(ctypeBuf.Bytes()))
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", post.Outfile))
			w.Header().Set("Content-Transfer-Encoding", "binary")
			w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
			w.WriteHeader(http.StatusOK)
			_, err = io.Copy(w, buf)
		}
		if err != nil {
			log.Errorf("error sending response to request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, err)
		} else {
			log.Infof("processed request ( %s %s ) from %s", r.Method, r.URL, r.RemoteAddr)
		}
	}
}

func UIPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Noticef("new request ( %s %s ) from %s", r.Method, r.URL, r.RemoteAddr)
		if DebugHTTPRequests {
			b, _ := httputil.DumpRequest(r, true)
			log.Debugf("request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, string(b))
		}
		if r.Method != http.MethodGet || r.URL.Path != "/" {
			log.Errorf("rejected request ( %s %s ) from %s: bad method or path", r.Method, r.URL, r.RemoteAddr)
			bye(w, r)
			return
		}
		tpl, _, err := FnMap.BuildHTMLTemplate(false, "ui", uiPage, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("error building template for response to request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, err)
			return
		}
		buf := new(bytes.Buffer)
		err = tpl.ExecuteTemplate(buf, "ui", struct{ Raddr string }{r.RemoteAddr})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("error rendering template for response to request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(buf.Bytes())
		//_, err := fmt.Fprintf(w, "%s\n%s", htmlHead, htmlForm)
		if err != nil {
			log.Errorf("error writing response to request ( %s %s ) from %s: %s", r.Method, r.URL, r.RemoteAddr, err)
		}
	}
}

const (
	uiPage       = htmlHead + htmlForm
	renderedPage = htmlHead + htmlArticle
	htmlHead     = `
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
	padding: 12px 20px;
	background: #eee;
	border-radius: 3px;
	border: 3px solid;
	box-shadow: 0 1px 2px rgba(0, 0, 0, .1);
	overflow:auto;
	display: -webkit-flex;
	-webkit-justify-content: center;
	-webkit-align-items: center;
	position: absolute;
	display: table;
}
textarea {
	display: inline-block;
	min-width: 25em;
	min-height: 15em;
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
	color: white;
	padding: 0.5rem;
	font-family: sans-serif;
	border-radius: 0.3rem;
	cursor: pointer;
	margin-top: 1rem;
}
.custom-file-label {
	content: "add templates..";
}
code {
    white-space: pre-wrap;
    overflow-wrap: break-word;
	overflow: auto;
}
</style>
<title>xprint render</title>
`
	htmlForm = `
<body>
<div class="container">
<form method="post" action="render" enctype="multipart/form-data" spellcheck="false">
    <p>
		<label for="text">TEMPLATE</label>
		<pre style="max-height: 50em; overflow: scroll;"><code class="codeblock"><textarea class="text" name="template" id="template">` + "{{`hello {{.client}}, it's {{timestamp}}`}}" + `
</textarea></code></pre>
	</p>
	<p>
		<label for="text">DATA</label>
		<pre style="max-height: 50em; overflow: scroll;"><code class="codeblock"><textarea class="text" name="data" id="data">{"client": "{{.Raddr}}"}</textarea></code></pre>
	</p>
	<p>
	<div class="col-md-offset-2 col-md-10 btn-group">
		<p>
			<input style="background-color:cyan" type="file" id="templates" name="templates" accept="text/*" multiple/>
			<label class="custom-file-label" for="templates">upload more templates to include</label>
		</p>

		<p>
			<input type="text" placeholder="rendered.out" name="outfile">
			<label for="outfile">output filename</label>

			<input type="checkbox" id="forcedl" name="forcedl" value="true">
			<label for="forcedl">force download</label>
		</p>
	

		<input style="background-color:green" type="submit" class="submit" value="Submit" />
		<input style="background-color:indigo" type="reset" value="Reset" class="btn btn-danger pull-right"/>
	</div>
	</p>
</form>
</div>
</body>
</html>
`
	htmlArticle = `
<div class="container">
<article class="all-browsers">
<pre style="max-height: 25em; overflow: scroll;"><code class="codeblock">{{.Output}}</code></pre>
</article>
</div>
</body>
</html>`
)

func bye(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://www.youtube.com/watch?v=dQw4w9WgXcQ?autoplay=1", http.StatusPermanentRedirect)
}
