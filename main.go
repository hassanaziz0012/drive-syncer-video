package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()

	conf := readConfig("config.json")

	drv := NewDrive(conf)
	drv.Connect(false, ctx)

	for _, f := range conf.Folders {
		UploadFolder(drv, f)
	}

	// r, err := drv.api.Files.List().PageSize(10).Do()
	// if err != nil {
	// 	log.Fatalf("Unable to fetch files from Drive: %v\n", err)
	// }
	// for _, f := range r.Files {
	// 	fmt.Println(f.Name)
	// }
}

func getClient(config *oauth2.Config, force_refresh bool) *http.Client {
	token_file := "token.json"

	var tok *oauth2.Token
	tok, err := tokenFromFile(token_file)
	if err != nil || force_refresh {
		tok = getTokenFromWeb(config)
		saveToken(token_file, tok)
	}

	return config.Client(context.Background(), tok)
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	json.NewDecoder(f).Decode(tok)

	return tok, nil
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following URL and grant necessary permissions. Then paste the authorization code below:\n%s\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v\n", err)
	}

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v\n", err)
	}

	return tok
}

func saveToken(path string, token *oauth2.Token) {
	log.Println("saving credential file to ", path)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600) // (owner: read/write, everyone else: no permissions)
	if err != nil {
		log.Fatalf("Unable to cache oauth2 token: %v\n", err)
	}
	defer f.Close()

	json.NewEncoder(f).Encode(token)
}

type Drive struct {
	config *Config
	api    *drive.Service
}

func NewDrive(c *Config) *Drive {
	return &Drive{config: c}
}

func (d *Drive) Connect(force_refresh bool, ctx context.Context) {
	b, err := os.ReadFile("client.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v\n", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file: %v\n", err)
	}

	client := getClient(config, force_refresh)

	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v\n", err)
	}

	d.api = srv
}
