package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/namsral/flag"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

//go:embed main.css main.js credentials.json
var files embed.FS

var activeRow = 2 // Used as a cursor to track what to show next. Start at 2 as header row expected.

func main() {

	var spreadsheet = flag.String("spreadsheet", "", "E.g. https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit.")
	var sheetID = flag.String("sheetID", "", "Sheet ID if not the first sheet in the document.")

	flag.Parse()

	if *spreadsheet == "" {
		fatalError("Missing required argument: spreadsheet.")
	}

	if *sheetID == "" {
		fatalError("Missing required argument: sheetID.")
	}

	spreadsheetID := spreadsheetID(*spreadsheet)
	svc := getSheetsService()

	checkHeaders(svc, spreadsheetID, *sheetID)

	http.HandleFunc("/queue", queueHandler(svc, spreadsheetID, *sheetID))
	http.HandleFunc("/proxy", proxyHandler())
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.FS(files))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println(fmt.Sprintf("Ready to diff! Go to: http://localhost:%s/queue.", port))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func queueHandler(svc *sheets.Service, spreadsheetID string, sheetID string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			// handle update and bump
			comment := req.FormValue("comment")
			if comment != "" { // rejection
				reject(svc, spreadsheetID, sheetID, activeRow, comment)
			} else {
				accept(svc, spreadsheetID, sheetID, activeRow)
			}

			_, nextURL, rowCount := getNextRow(svc, spreadsheetID, sheetID, activeRow)
			activeRow = rowCount
			resp.Write([]byte(queuePage(nextURL)))
		case http.MethodGet:
			_, nextURL, rowCount := getNextRow(svc, spreadsheetID, sheetID, activeRow)
			activeRow = rowCount
			resp.Write([]byte(queuePage(nextURL)))
		default:
			resp.WriteHeader(http.StatusNotFound)
		}
	}
}

func proxyHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guardianURL := r.URL.Query().Get("url")
		resp, err := http.Get(guardianURL)
		if err != nil {
			fatalError(fmt.Sprintf("Unable to fetch: %s. %s", guardianURL, err.Error()))
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		w.Write(body)
	}
}

func fatalError(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func spreadsheetID(url string) string {
	re := regexp.MustCompile("https://docs.google.com/spreadsheets/d/(.*)/edit")
	matches := re.FindStringSubmatch(url)
	if matches == nil {
		fatalError("Spreadsheet URL invalid: " + url)
	}

	return matches[1]
}

func checkHeaders(svc *sheets.Service, spreadsheetID string, sheetID string) {
	headersRange := fmt.Sprintf("%s!A1:C1", sheetID)
	resp := getRange(svc, spreadsheetID, headersRange)
	headers := resp.Values[0]

	if headers[0] != "URL" || headers[1] != "Status" || headers[2] != "Comment" {
		fatalError("Invalid sheet first three column headings. Must be: URL, Status, Comment.")
	}
}

func getNextRow(svc *sheets.Service, spreadsheetID string, sheetID string, activeRow int) (bool, string, int) {
	startRow := activeRow + 1
	allRange := fmt.Sprintf("%s!A%d:C", sheetID, startRow) // gets all cells
	cells := getRange(svc, spreadsheetID, allRange)

	for row, data := range cells.Values {
		if len(data) == 1 { // has URL but nothing else
			return true, fmt.Sprint(data[0]), row + startRow
		}
	}

	return false, "", 0
}

func reject(svc *sheets.Service, spreadsheetID string, sheetID string, activeRow int, comment string) {
	cellRange := fmt.Sprintf("%s!B%d:C%d", sheetID, activeRow, activeRow)
	values := [][]interface{}{{"Rejected", comment}}
	valueRange := sheets.ValueRange{
		Values: values,
	}

	_, err := svc.Spreadsheets.Values.Update(spreadsheetID, cellRange, &valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		fatalError(fmt.Sprintf("Unable to update sheet for active row %d: %v.", activeRow, err))
	}
}

func accept(svc *sheets.Service, spreadsheetID string, sheetID string, activeRow int) {
	cellRange := fmt.Sprintf("%s!B%d", sheetID, activeRow)
	values := [][]interface{}{{"Accepted"}}
	valueRange := sheets.ValueRange{
		Values: values,
	}

	_, err := svc.Spreadsheets.Values.Update(spreadsheetID, cellRange, &valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		fatalError(fmt.Sprintf("Unable to update sheet for active row %d: %v.", activeRow, err))
	}
}

func getCell(svc *sheets.Service, spreadsheetID string, sheetID string, col string, row int) (bool, string) {
	cellRange := fmt.Sprintf("%s!%s%d", sheetID, col, row)
	resp := getRange(svc, spreadsheetID, cellRange)

	if len(resp.Values) == 0 {
		return false, ""
	}

	return true, fmt.Sprint(resp.Values[0][0])
}

func getRange(svc *sheets.Service, spreadsheetID string, readRange string) *sheets.ValueRange {
	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		fatalError(fmt.Sprintf("Unable to retrieve data from sheet: %v.", err))
	}

	return resp
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getSheetsService() *sheets.Service {
	ctx := context.Background()
	b, err := files.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	return srv
}
