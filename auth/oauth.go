package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/hassanaziz0012/drive-syncer-video/internal/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func getClient(config *oauth2.Config, force_refresh bool) *http.Client {
	token_file := "token.json"

	var tok *oauth2.Token
	tok, err := tokenFromFile(token_file)
	if err != nil || force_refresh {
		tok = getTokenFromWeb(config)
		saveToken(token_file, tok)
	}

	ts := config.TokenSource(context.Background(), tok)

	tok, err = ts.Token()
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "token has been expired") {
			tok = getTokenFromWeb(config)
		} else {
			log.Fatalf("failed to generate token: %v\n", err)
		}
	}
	saveToken(token_file, tok)

	return oauth2.NewClient(context.Background(), ts)
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

func NewDrive(c *models.Config) *models.Drive {
	return &models.Drive{Config: c}
}

type APIAccessStatus int

const (
	AccessExpired APIAccessStatus = iota
	AccessGranted
	AccessRejected
)

func testAccess(drv *models.Drive) APIAccessStatus {
	_, err := drv.Api.Files.List().PageSize(1).Do()
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "token expired") || strings.Contains(strings.ToLower(err.Error()), "token has been expired") {
			log.Printf("token has expired.\n")
			return AccessExpired
		} else {
			log.Printf("token has been rejected.\n")
			return AccessRejected
		}
	}

	log.Printf("token is valid.\n")
	return AccessGranted
}

func CreateDriveService(force_refresh bool) *drive.Service {
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
	return srv
}

func ConnectToDrive(d *models.Drive, force_refresh bool, ctx context.Context) {
	srv := CreateDriveService(force_refresh)
	d.Api = srv

	status := testAccess(d)
	if status == AccessExpired && !force_refresh {
		srv := CreateDriveService(true)
		d.Api = srv
	}
}
