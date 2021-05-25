package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

type TplData struct {
	FrontendURL string
	DCRURL      string
	Width       int
	Height      int
	Evergreens  []string
	Latests     []CAPIItem
}

type CAPIItem struct {
	WebURL string `json:"webUrl"`
	ApiURL string `json:"apiUrl"`
}

type CAPIResponse struct {
	Response struct {
		Results []CAPIItem `json:"results"`
	} `json:"response"`
}

//go:embed main.css main.js
var files embed.FS

var breakpoints = map[string]int{
	"mobile":  480,
	"leftCol": 1140,
}

var evergreens = []string{
	"https://www.theguardian.com/education/ng-interactive/2020/sep/05/the-best-uk-universities-2021-league-table",
	"https://www.theguardian.com/world/ng-interactive/2020/nov/18/colette-a-former-french-resistance-member-confronts-a-family-tragedy-75-years-later",
	"https://www.theguardian.com/football/ng-interactive/2020/dec/21/the-100-best-male-footballers-in-the-world-2020",
	"https://www.theguardian.com/help/ng-interactive/2017/mar/17/contact-the-guardian-securely",
}

// TODO memoise?
func latestInteractives() ([]CAPIItem, error) {
	reqURL := "https://content.guardianapis.com/search?api-key=test&type=interactive&page-size=20"
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	data := CAPIResponse{}
	err = json.Unmarshal(body, &data)

	return data.Response.Results, err
}

func main() {

	isLocalDCR := flag.Bool("local", false, "If present, will use local DCR (on port 3030) for comparison rather than PROD.")
	flag.Parse()

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.FS(files))))

	http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {
		guardianURL := r.URL.Query().Get("url")
		resp, err := http.Get(guardianURL)
		check(w, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		w.Write(body)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		markup := `
<!DOCTYPE html>
<html>
	<head>
	<meta charset="utf-8">
	<title>Interactive DCR diff</title>
	<link rel="icon" href="https://static.guim.co.uk/images/favicon-32x32-dev-yellow.ico">
	<link href="/static/main.css" rel="stylesheet">
	</head>
	<body>
		<h1>Interactive DCR diff tool</h1>
		
		<form>
			<label for="url">Custom URL:</label>
			<input type="url" id="url" name="target" value="{{.FrontendURL}}">
			<button>Update</button>
		</form>

		<form>
			<label for="urls">Example URLs:</label>
			<select name="target">
				<optgroup label="Evergreen interactives">
					{{range .Evergreens}}
						<option value="{{.}}">{{.}}</option>
					{{end}}
				</optgroup>
			
				<optgroup label="Latest interactives">
					{{range .Latests}}
						<option value="{{.WebURL}}">{{.WebURL}}</option>
					{{end}}
				</optgroup>
			</select>
		
			<button>Update</button>
		</form>


		<div class="iframes">
		<iframe id="interactive-frontend"
    		title="Inline Frame Example"
    		width="{{.Width}}"
    		height="{{.Height}}"
    		src="/proxy?url={{.FrontendURL}}">
		</iframe>
		<iframe id="interactive-dcr"
			title="Inline Frame Example"
			width="{{.Width}}"
			height="{{.Height}}"
			src="/proxy?url={{.DCRURL}}">
		</iframe>
	</body>
</html>			
		`

		t, err := template.New("diff").Parse(markup)
		check(w, err)

		latests, err := latestInteractives()
		check(w, err)

		target := "https://www.theguardian.com/education/ng-interactive/2020/sep/05/the-best-uk-universities-2021-league-table"
		customTarget := r.URL.Query().Get("target")
		if customTarget != "" {
			target = customTarget
		}

		targetURL, _ := url.Parse(target)
		path := targetURL.Path

		frontendTarget := fmt.Sprintf("https://www.theguardian.com%s", path)
		DCRTarget := ""
		if *isLocalDCR {
			DCRTarget = fmt.Sprintf("http://localhost:3030/Interactive?url=https://www.theguardian.com%s", path)
		} else {
			DCRTarget = frontendTarget + "?dcr"
		}

		data := TplData{
			FrontendURL: frontendTarget,
			DCRURL:      DCRTarget,
			Width:       breakpoints["mobile"],
			Height:      breakpoints["mobile"] * 3,
			Evergreens:  evergreens,
			Latests:     latests,
		}

		buf := bytes.Buffer{}
		err = t.Execute(&buf, data)
		check(w, err)

		w.Write(buf.Bytes())
	})

	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func check(w http.ResponseWriter, err error) {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
}
