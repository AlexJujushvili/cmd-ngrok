package main

import (
	"context"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"golang.ngrok.com/ngrok/v2"
)

var (
	activeCmd  *exec.Cmd
	cmdMu      sync.Mutex
	currentDir string
	agent      ngrok.Agent
	ln         net.Listener // დავამატეთ გლობალური ლისენერი
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

	agent, err = ngrok.NewAgent(
		ngrok.WithAuthtoken("36gmx4MIeEG3BrAIUf9RiRC8Dzg_3g7KHHphRsiWQzNsQcJXu"),
	)
	if err != nil {
		fmt.Println("Agent error:", err)
		return
	}

	// ვინახავთ ლისენერს გლობალურ ცვლადში 'ln'
	ln, err = agent.Listen(
		ctx,
		ngrok.WithURL("liked-together-mantis.ngrok-free.app"),
	)
	if err != nil {
		fmt.Println("Listener error:", err)
		return
	}

	fmt.Printf("Terminal Online: %s\n", ln.Addr().String())
	
	// სერვერი გაეშვება ლისენერზე
	err = http.Serve(ln, http.HandlerFunc(handler))
	if err != nil {
		// როცა ln.Close() მოხდება, Serve დააბრუნებს შეცდომას, რაც ნორმალურია
		fmt.Println("Server stopped:", err)
	}
}

func StopNgrok() {
	fmt.Println("Shutting down...")

	// 1. ჯერ ვხურავთ ლისენერს (ეს გათიშავს http.Serve-ს)
	if ln != nil {
		ln.Close()
	}

	// 2. შემდეგ ვთიშავთ აგენტს
	if agent != nil {
		agent.Disconnect()
		agent = nil
	}
	
	fmt.Println("Ngrok server fully stopped")
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tpl, _ := template.New("tpl").Parse(html_tpl)
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
			
			// ვიყენებთ goroutine-ს, რომ პასუხის გაგზავნა მოესწროს
			go StopNgrok()
			return
		}

		runProcess(w, command)
	}
}

// runProcess და killProcess ფუნქციები რჩება უცვლელი
func runProcess(w http.ResponseWriter, cmdStr string) {
	cmdStr = strings.TrimSpace(cmdStr)
	args := strings.Fields(cmdStr)
	if len(args) == 0 { return }

	if args[0] == "cd" {
		if len(args) < 2 {
			w.Write([]byte(currentDir))
			return
		}
		target := args[1]
		newPath := filepath.Join(currentDir, target)
		if filepath.IsAbs(target) { newPath = target }

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

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}

	cmd.Dir = currentDir
	activeCmd = cmd
	cmdMu.Unlock()

	out, err := cmd.CombinedOutput()

	cmdMu.Lock()
	activeCmd = nil
	cmdMu.Unlock()

	if err != nil && out == nil {
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

const html_tpl = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Terminal - Stable Core V11</title>
    <link href="https://fonts.googleapis.com/css?family=Roboto+Mono&display=swap" rel="stylesheet">
    <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=0">
    <style>
        *, *::before, *::after { box-sizing: border-box; font-family: 'Roboto Mono', monospace; }
        :focus { outline: none; }
        body { margin: 0; overflow: hidden; background-color: #333444; }
        #app { height: 100vh; display: flex; justify-content: center; align-items: center; }
        #terminal { width: 95vw; max-width: 900px; height: 85vh; box-shadow: 0 20px 50px rgba(0,0,0,0.5); border-radius: 8px; overflow: hidden; display: flex; flex-direction: column; background-color: #222333; }
        #window { height: 40px; display: flex; align-items: center; padding: 0 15px; background-color: #222345; color: #F4F4F4; flex-shrink: 0; }
        #field { flex-grow: 1; padding: 20px; overflow-y: auto; color: #F4F4F4; position: relative; scroll-behavior: smooth; }
        #query { color: #2ECC40; font-weight: bold; margin-right: 10px; }
        .line { margin-bottom: 8px; white-space: pre-wrap; word-break: break-all; line-height: 1.5; font-size: 14px; position: relative; min-height: 20px; }
        #active-input { position: absolute; left: 0; top: 0; width: 100%; height: 100%; opacity: 0; border: none; background: transparent; z-index: 2; cursor: text; }
        #cursor { display: inline-block; width: 8px; height: 15px; background-color: #F4F4F4; animation: 1s blink step-end infinite; vertical-align: middle; }
        @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }
    </style>
</head>
<body>
    <div id="root"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/react/16.12.0/umd/react.production.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/react-dom/16.11.0/umd/react-dom.production.min.js"></script>

    <script>
        const { Component } = React;

        class Field extends Component {
            constructor(props) {
                super(props);
                this.state = {
                    fieldHistory: [{ text: 'Terminal v11.0 Online' }],
                    userInput: ''
                };
            }

            componentDidMount() { if (this.inputRef) this.inputRef.focus(); }

            componentDidUpdate(prevProps, prevState) {
                if (prevState.fieldHistory.length !== this.state.fieldHistory.length && this.activeLineRef) {
                    this.activeLineRef.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
            }

            handleInputExecution(cmd) {
                const cleanCmd = cmd.trim();
                if (!cleanCmd || cleanCmd === 'cls') {
                    if (cleanCmd === 'cls') this.setState({ fieldHistory: [] });
                    return;
                }
                
                const params = new URLSearchParams();
                params.append('cmd', cleanCmd);
                fetch('/', { method: 'POST', body: params })
                    .then(r => r.text())
                    .then(result => {
                        // სერვერის გათიშვის ლოგიკა
                        if (result.includes('SERVER_STOPPED')) {
                            document.body.innerHTML = '<div style="height:100vh; display:flex; flex-direction:column; justify-content:center; align-items:center; background-color:#f0f0f0; font-family:sans-serif; text-align:center;">' +
                                '<div style="font-size:50px; margin-bottom:20px;">OFF</div>' +
                                '<div style="font-size:24px; color:#333; font-weight:bold;">სერვერი გაითიშა. კავშირი გაწყვეტილია.</div>' +
                                '</div>';
                            return;
                        }
                        this.setState(s => ({ fieldHistory: [...s.fieldHistory, { text: result || 'Done.' }] }));
                    })
                    .catch(() => {
                        this.setState(s => ({ fieldHistory: [...s.fieldHistory, { text: 'კავშირი გაწყდა.' }] }));
                    });
            }

            render() {
                return React.createElement('div', { id: 'field', onClick: () => this.inputRef && this.inputRef.focus() },
                    this.state.fieldHistory.map((l, i) => React.createElement('div', { key: i, className: 'line' },
                        l.isCommand && React.createElement('span', { id: 'query' }, 'RT C:\\>'),
                        React.createElement('span', null, l.text)
                    )),
                    React.createElement('div', { className: 'line', ref: el => this.activeLineRef = el },
                        React.createElement('span', { id: 'query' }, 'RT C:\\>'),
                        React.createElement('span', null, this.state.userInput),
                        React.createElement('div', { id: 'cursor' }),
                        React.createElement('input', {
                            ref: el => this.inputRef = el,
                            id: 'active-input',
                            value: this.state.userInput,
                            onChange: e => this.setState({ userInput: e.target.value }),
                            onKeyDown: e => { if (e.key === 'Enter') { 
                                const val = this.state.userInput;
                                this.setState(s => ({ fieldHistory: [...s.fieldHistory, { text: val, isCommand: true }], userInput: '' }));
                                this.handleInputExecution(val);
                            }},
                            autoComplete: 'off',
                            spellCheck: 'false'
                        })
                    )
                );
            }
        }

        const App = () => React.createElement('div', { id: 'app' },
            React.createElement('div', { id: 'terminal' },
                React.createElement('div', { id: 'window' },
                    React.createElement('div', { style: { display: 'flex', gap: '5px' } },
                        React.createElement('div', { style: { width: '12px', height: '12px', borderRadius: '50%', backgroundColor: '#ff5f56' } }),
                        React.createElement('div', { style: { width: '12px', height: '12px', borderRadius: '50%', backgroundColor: '#ffbd2e' } }),
                        React.createElement('div', { style: { width: '12px', height: '12px', borderRadius: '50%', backgroundColor: '#27c93f' } })
                    ),
                    React.createElement('span', { style: { marginLeft: 'auto', fontSize: '12px', opacity: 0.5 } }, 'STABLE_CORE_V11')
                ),
                React.createElement(Field)
            )
        );

        ReactDOM.render(React.createElement(App), document.querySelector('#root'));
    </script>
</body>
</html>`








