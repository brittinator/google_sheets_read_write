package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("sheets.googleapis.com-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func readSheet(srv *sheets.Service, sheetID, readRange string) (resp *sheets.ValueRange, err error) {
	resp, err = srv.Spreadsheets.Values.Get(sheetID, readRange).Do()
	return resp, nil
}

// appendData adds new cells after the last row with data in a sheet,
func appendData(srv *sheets.Service, sheetID, writeRange string, data [][]interface{}) error {
	valueRange := sheets.ValueRange{MajorDimension: "ROWS", Values: data}
	if _, err := srv.Spreadsheets.Values.Update(sheetID, writeRange, &valueRange).ValueInputOption("USER_ENTERED").Do(); err != nil {
		return err
	}
	return nil
}

func connectSheetsClient() (*sheets.Service, error) {
	var srv *sheets.Service
	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/sheets.googleapis.com-go-quickstart.json
	// list of scope here: https://developers.google.com/identity/protocols/googlescopes
	// config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return srv, err
	}
	client := getClient(ctx, config)

	srv, err = sheets.New(client)
	return srv, err
}

var sheetID string

func init() {
	flag.StringVar(&sheetID, "id", "", "found in the URL of the sheet")
}

func main() {
	flag.Parse()
	// !!! Currently flag not setup properly
	fmt.Printf("id: %v\n", sheetID)

	fmt.Println("Connecting to Sheets API")
	srv, err := connectSheetsClient()
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}
	spreadsheetID := sheetID
	// readRange automatically only captures filled cells
	readRange := "ClassData!A2:B"

	fmt.Println("reading spreadsheet")
	resp, err := readSheet(srv, spreadsheetID, readRange)
	fmt.Println(resp)
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet. %v", err)
	}
	lastIndex := len(resp.Values)
	fmt.Printf("last Row: %v", lastIndex)
	if len(resp.Values) > 0 {
		fmt.Println("Name, Activity:")
		for i, row := range resp.Values {
			fmt.Println(i)
			// Print columns A and E, which correspond to indices 0 and 4.
			fmt.Printf("%s, %s\n", row[0], row[1])
		}
	} else {
		fmt.Print("No data found.")
	}

	writeRange := "ClassData!" + "A" + strconv.Itoa(lastIndex+2) + ":B"
	var data [][]interface{}
	var row []interface{}
	row = append(row, time.Now().Format("1/02/2006"))
	row = append(row, "squash")
	data = append(data, row)
	writeErr := appendData(srv, spreadsheetID, writeRange, data)
	if writeErr != nil {
		log.Fatalf("Unable to append data to sheet, %v", writeErr)
	}
}
