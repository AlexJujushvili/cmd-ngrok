package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"time"

	"golang.ngrok.com/ngrok/v2"
)

var (
	server    *http.Server
	agent     ngrok.Agent
	url       string
	camstatus string
	clientIP  string
)

func main() {
	startNgrok()

}

func startNgrok() {

	ctx := context.Background()
	var err error

	agent, err = ngrok.NewAgent(
		//ngrok.WithAuthtoken("YOUR_AUTHTOKEN_HERE"),
		ngrok.WithAuthtoken("36gmx4MIeEG3BrAIUf9RiRC8Dzg_3g7KHHphRsiWQzNsQcJXu"),
	)
	if err != nil {
		println("Agent error:", err)
		return
	}

	listener, err := agent.Listen(
		ctx,
		ngrok.WithURL("liked-together-mantis.ngrok-free.app"),
		ngrok.WithDescription("ტესტი"),
	)
	if err != nil {
		println("Listener error:", err)
		return
	}

	url = listener.URL().String()
	println("HTTP endpoint online: %s", url)

	server = &http.Server{
		Handler: http.HandlerFunc(handler),
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		println("Server error:", err)
	}
}

func stopNgrok() {

	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		server = nil
	}
	if agent != nil {
		agent.Disconnect()
		agent = nil
	}
	println("Ngrok server stopped")
}

// ---- Handlers ----
func handler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		//http.ServeFile(w, r, "index.html")
		handleDirectory(w, r)

	case "POST":
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
			return
		}

		// მონაცემების მიღება პოსტის მეთოდით
		command := r.Form.Get("cmd")

		fmt.Println("მივიღე:", command)

		// პასუხი კლიენტს -სტატუსი
		w.WriteHeader(http.StatusOK)

		//w.Write([]byte("აქ პასუხი სტრინგით - სტრინგი1\n"))

		cmd1(w, command)

	default:
		w.Write([]byte("მხარსუჭერს მხოლოდ გეთ-ს და პოსტ_ს\n"))

	}
}

func cmd1(w http.ResponseWriter, cmdrun string) string {
	//cmd := exec.Command("cmd", "/C", "ping mail.ru")
	//out, err := exec.Command("cmd", "/C", "ipconfig").Output()

	out, err := exec.Command(cmdrun).Output()
	//w.Write(out)
	w.Write([]byte(out))

	if err != nil {

		w.Write([]byte("is not recognized as an internal or external command operable program or batch file"))

		return ("")
	}
	fmt.Printf("The date is %s\n", out)

	return ""
}

//////////// http post/get method ///////////////

// /////////////////////////////////////////////////
func handleDirectory(w http.ResponseWriter, r *http.Request) {

	tpl, err := template.New("tpl").Parse(html_tpl)
	if err != nil {
		http.Error(w, "500 შიდა სეცდომა: არიკითხე ვებ გვერდი.", 500)
		fmt.Println(err)
		return
	}

	data := ""

	err = tpl.Execute(w, data)
	if err != nil {
		fmt.Println(err)
	}
}

// ////////////////////////////

const html_tpl = `<!DOCTYPE html>
<html lang="en" >
<head>
  <meta charset="UTF-8">
  <title>RCMD</title>
  <meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="description" content="enter a command.">

    <title>RCMD</title>

<link rel="icon" href="https://unilogue.github.io/favi.png">

  
  
  <script type="text/javascript" src="https://unilogue.github.io/js/jquery-1.11.3.min.js"></script>


  <style type="text/css">
  ::-webkit-scrollbar {
      display: none;
    }
    /* fix double style of selecting text in terminal */
    
    .terminal-output span,
    .terminal-output a,
    .cmd div,
    .cmd span,
    .terminal td,
    .terminal pre,
    .terminal h1,
    .terminal h2,
    .terminal h3,
    .terminal h4,
    .terminal h5,
    .terminal h6 {
      -webkit-touch-callout: initial;
      -webkit-user-select: initial;
      -khtml-user-select: initial;
      -moz-user-select: initial;
      -ms-user-select: initial;
      user-select: initial;
    }
    
    .terminal,
    .terminal-output,
    .terminal-output div {
      -webkit-touch-callout: initial;
      -webkit-user-select: initial;
      -khtml-user-select: initial;
      -moz-user-select: initial;
      -ms-user-select: initial;
      user-select: initial;
    }
    /* firefox hack */
    
    @-moz-document url-prefix() {
      .terminal,
      .terminal-output,
      .terminal-output div {
        -webkit-touch-callout: initial;
        -webkit-user-select: initial;
        -khtml-user-select: initial;
        -moz-user-select: initial;
        -ms-user-select: initial;
        user-select: initial;
      }
    }
    
    p {
      font-family: monospace;
      text-align: justify;
      color: #aaa;
      font-size: 12px;
      line-height: 14px;
      text-rendering: optimizeLegibility;
    }
    
    li {
      font-family: monospace;
      text-align: justify;
      color: #aaa;
      font-size: 12px;
      line-height: 14px;
      text-rendering: optimizeLegibility;
    }
    
    ul, ol {
      list-style-type: none;
    }
    
    li:before {
      content: '> '
    }
    
    body {
      background-color: #000;
    }
    
    body p {
      font-family: monospace;
      text-align: justify;
      color: #aaa;
      font-size: 12px;
      line-height: 14px;
      text-rendering: optimizeLegibility;
    }
    
    body a {
      text-decoration: none;
      color: #aaa;
      font-family: monospace;
    }
    
    body a:before {
      color: #aaa;
      content: '<a href="';
    }
    
    body a:after {
      color: #aaa;
      content: '">';
    }
    
    sup a:before,
    footer a:before {
      color: #aaa;
      content: '';
    }
    
    sup a:after,
    footer a:after {
      color: #aaa;
      content: '';
    }
    
    h1 {
      font-family: monospace;
      text-align: justify;
      color: #aaa;
      font-size: 36px;
      line-height: 36px;
      text-rendering: optimizeLegibility;
    }
    
    h2 {
      font-family: monospace;
      text-align: justify;
      color: #aaa;
      font-size: 24px;
      line-height: 24px;
      text-rendering: optimizeLegibility;
    }
    
    hr {
      border: 0;
      border-top: 1px solid #aaa;
      height: 0;
    }
    
    pre {
      font-family: monospace;
      color: #aaa;
      margin-left: 50px;
      font-size: 14px;
      line-height: 16px;
      white-space: pre-wrap;
      text-rendering: optimizeLegibility;
    }
    
    blockquote {
      font-family: monospace;
      text-align: justify;
      color: #aaa;
      font-size: 18px;
      padding: 5px;
      text-rendering: optimizeLegibility;
    }
    
    .datamap path.datamaps-graticule {
      pointer-events: all;
    }
    
    .datamap path.datamaps-subunit {
      pointer-events: all;
    }
</style>

</head>
<body>
<!-- partial:index.partial.html -->
<!DOCTYPE html>
<html lang="en">

<head>
  <meta charset="utf-8">
  <title>RCMD</title>
  <!-- <link rel="icon" href="//unilogue.github.io/favi.png"> -->
  <link href="//unilogue.github.io/jquery.terminal/css/jquery.terminal.css" rel="stylesheet">
  <!-- <link rel="stylesheet" href="//code.ionicframework.com/ionicons/2.0.1/css/ionicons.min.css"> -->
  <!-- <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/font-awesome/4.4.0/css/font-awesome.min.css"> -->
  
  <script src="//unilogue.github.io/jquery.terminal/js/jquery-1.7.1.min.js"></script>
  <!-- <script src="//unilogue.github.io/jquery.terminal/js/jquery.mousewheel-min.js"></script> -->
 <script src="//unilogue.github.io/jquery.terminal/js/jquery.terminal-min.js"></script>

  
</head>

<body>
  <div id="cmd">
  <p>Alex JujuSvili
    <br>(c) 2023</p>
  </div>
</body>
</html>
<!-- partial -->
  
<!--ტერმიონალის სკრიპტი -->
  <script> 
$('#cmd').terminal(function(command, term) {
  
  // http post მეთოდი
const data = new URLSearchParams();
    data.append("cmd", command);
    

    fetch('/', {
        method: 'POST',
        body: data,
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
    })
    .then(response => response.text())
    .then(data => {
        console.log('Response from server:', data);
        
        term.echo (data, {raw:false});
        
    })
    .catch(error => {
        console.error('Error sending POST request:', error);
    });
    term.echo ('<div id="map"></div>', {raw:true});

    // http post მეთოდი
  
  
  if (command == 'i') {
    term.echo ('დროებით არ მუშაობს.', {raw:true});

  } else if (command == 'd') {
 term.echo ('<div id="derive"></div>', {raw:true});
  } else if (command == 'g') {
 term.echo ('<div id="glossary"></div>' , {raw:true}); 
  } else if (command == 'git') {
    window.open('https://github.com', '_blank');
  } else if (command == '?') {
    term.echo('COMMANDS: [i]ipconfig\t\t[d]erive\t[g]lossary\r\n\t\t  [?] help\t [clear]\t [git]hub');
  } ///else if (command != '') {
    //term.echo('Command not found!');
  //}
}, 

{
  greetings: 'COMMANDS: [i]pcongig\t\t[d]erive\t[g]lossary\r\n\t\t  [?] help\t [clear]\t [git]hub',
  name: 'A751',
  prompt: '> '
});
</script>

</body>
</html>

`
