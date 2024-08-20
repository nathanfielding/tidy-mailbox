package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/pflag"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func main() {
	creds := pflag.StringP("credentials", "c", "credentials.json", "File path to credentials file")
	tokPath := pflag.StringP("token", "t", "token.json", "File path to token file")

	flag.Parse()

	ctx := context.Background()
	b, err := os.ReadFile(*creds)
	if err != nil {
		log.Fatalf("Unable to read secret file: %v", err)
	}

	cfg, err := google.ConfigFromJSON(b, gmail.MailGoogleComScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(*tokPath, cfg)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	if err := deleteSpam(srv); err != nil {
		log.Fatalf("Unable to delete contents of spam folder: %v", err)
	}
}

func getClient(tokPath string, config *oauth2.Config) *http.Client {
	tok, err := tokenFromFile(tokPath)
	if err != nil {
		tok = tokenFromWeb(config)
		if err := saveToken(tokPath, tok); err != nil {
			return nil
		}
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
	err = json.NewDecoder(f).Decode(tok)

	return tok, err
}

func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving OAuth token to :%s\n", path)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Printf("Unable cache OAuth token: %v", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

func tokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from the web: %v", err)
	}

	return tok
}

func deleteSpam(srv *gmail.Service) error {
	user, query := "me", "label:Spam"
	spam, err := srv.Users.Messages.List(user).Q(query).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve spam: %v", err)
	}

	ids := []string{}
	for _, s := range spam.Messages {
		ids = append(ids, s.Id)
	}
	batchDelReq := &gmail.BatchDeleteMessagesRequest{
		Ids: ids,
	}

	if err := srv.Users.Messages.BatchDelete(user, batchDelReq).Do(); err != nil {
		return fmt.Errorf("failed to perform deletion: %v", err)
	}

	return nil
}
