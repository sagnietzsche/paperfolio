package main

import (
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets/pdfjs/pdf.min.mjs
var bundledPDFJS []byte

//go:embed assets/pdfjs/pdf.worker.min.mjs
var bundledPDFJSWorker []byte

const pdfViewerHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Paperfolio PDF Viewer</title>
<style>
html,body{height:100%;margin:0;background:#202124;color:#eee;font:14px system-ui,sans-serif;overflow:hidden}
#toolbar{height:44px;display:flex;align-items:center;gap:8px;padding:0 12px;background:#303134;box-sizing:border-box}
button,input{background:#45464a;color:#fff;border:1px solid #65666a;border-radius:4px;padding:5px 8px}button{cursor:pointer}input{width:54px;text-align:center}
#pages{height:calc(100% - 44px);overflow:auto;text-align:center;padding:18px;box-sizing:border-box}
canvas{display:block;margin:0 auto 18px;background:#fff;box-shadow:0 2px 10px #0008;max-width:100%}
#status{margin-left:auto;color:#bbb}
</style>
</head>
<body>
<div id="toolbar"><button id="prev">Previous</button><button id="next">Next</button><span>Page</span><input id="page" type="number" min="1" value="1"><button id="zoomOut">−</button><button id="zoomIn">+</button><button id="fit">Fit</button><button id="search">Search</button><span id="status">Loading…</span></div>
<div id="pages"></div>
<script type="module">
import * as pdfjsLib from '/pdfjs/pdf.min.mjs';
<script>
(() => {
 const params=new URLSearchParams(location.search), pdfURL=params.get('pdf');
 const pages=document.getElementById('pages'), status=document.getElementById('status'), pageInput=document.getElementById('page');
 let pdf=null,current=1,scale=1.25;
 pdfjsLib.GlobalWorkerOptions.workerSrc='/pdfjs/pdf.worker.min.mjs';
 function render(n){if(!pdf||n<1||n>pdf.numPages)return; current=n; pageInput.value=n; pdf.getPage(n).then(page=>{const viewport=page.getViewport({scale});const canvas=document.createElement('canvas');canvas.width=viewport.width;canvas.height=viewport.height;canvas.dataset.page=n;return page.render({canvasContext:canvas.getContext('2d'),viewport}).promise.then(()=>{pages.replaceChildren(canvas);status.textContent='Page '+n+' of '+pdf.numPages;});}).catch(e=>status.textContent=e.message)}
 document.getElementById('prev').onclick=()=>render(current-1); document.getElementById('next').onclick=()=>render(current+1);
 pageInput.onchange=()=>render(Number(pageInput.value)); document.getElementById('zoomIn').onclick=()=>{scale=Math.min(scale+0.2,4);render(current)}; document.getElementById('zoomOut').onclick=()=>{scale=Math.max(scale-0.2,0.4);render(current)};
 document.getElementById('fit').onclick=()=>{scale=Math.max(0.4,Math.min(2.5,(pages.clientWidth-36)/612));render(current)};
 document.getElementById('search').onclick=()=>{const q=prompt('Search is available in the document text layer in the full PDF.js viewer. Enter a term to search:');if(q)alert('Search request: '+q+'\nUse the PDF.js viewer build for full text search.')};	if(!pdfURL){status.textContent='No PDF selected';return} pdfjsLib.getDocument(pdfURL).promise.then(x=>{pdf=x;render(1)}).catch(e=>{status.textContent='Unable to render PDF: '+e.message});

})();
</script></body></html>`

type pdfViewerServer struct {
	server *http.Server
	base   string
	token  string
}

func startPDFViewerServer(store *Store, paper *Paper) (*pdfViewerServer, error) {
	path, err := store.PDFAbsolutePath(paper)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(path)
	token, err := newID()
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(pdfViewerHTML))
	})
	mux.HandleFunc("/pdf", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != name || r.URL.Query().Get("token") != token {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	})
	mux.HandleFunc("/pdfjs/pdf.min.mjs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write(bundledPDFJS)
	})
	mux.HandleFunc("/pdfjs/pdf.worker.min.mjs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write(bundledPDFJSWorker)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start PDF viewer server: %w", err)
	}
	server := &http.Server{Handler: mux}
	result := &pdfViewerServer{server: server, base: "http://" + listener.Addr().String(), token: token}
	go func() { _ = server.Serve(listener) }()
	return result, nil
}

func (s *pdfViewerServer) documentURL(name string) (*url.URL, error) {
	return url.Parse(s.base + "/?pdf=" + url.QueryEscape(s.base+"/pdf?name="+url.QueryEscape(filepath.Base(name))+"&token="+url.QueryEscape(s.token)))
}

func (s *pdfViewerServer) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Close()
}

func (s *Store) PDFAbsolutePath(paper *Paper) (string, error) {
	if paper == nil || paper.PDFPath == nil || strings.TrimSpace(*paper.PDFPath) == "" {
		return "", fmt.Errorf("paper has no PDF")
	}
	uploads, err := filepath.Abs(s.UploadsDir)
	if err != nil {
		return "", fmt.Errorf("resolve uploads directory: %w", err)
	}
	candidate, err := filepath.Abs(filepath.Join(s.UploadsDir, filepath.Base(*paper.PDFPath)))
	if err != nil {
		return "", fmt.Errorf("resolve PDF path: %w", err)
	}
	rel, err := filepath.Rel(uploads, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("PDF path is outside the uploads directory")
	}
	info, err := os.Stat(candidate)
	if err != nil {
		return "", fmt.Errorf("find PDF: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("PDF path is a directory")
	}
	return candidate, nil
}

func fileURL(path string) (*url.URL, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}, nil
}
