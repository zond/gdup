package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
)

func main() {
	file := flag.String("file", "", "File to upload.")
	parent := flag.String("parent", "", "Google Drive ID of parent directory. Provide either -parent or -id.")
	id := flag.String("id", "", "ID of file to upload to. Provide either -parent or -id.")
	mime := flag.String("mime", "application/octet-stream", "MIME type of uploaded file.")
	flag.Parse()

	driveConfig := &oauth2.Config{
		ClientID:     "944095575246-jq2jufr9k7s244jl9qb4nk1s36av4cd5.apps.googleusercontent.com",
		ClientSecret: "U0Okcw5_XHz8565QPRsi1Nun",
		Scopes:       []string{"https://www.googleapis.com/auth/drive"},
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://accounts.google.com/o/oauth2/token",
		},
	}

	if *file == "" || (*parent == "" && *id == "") || (*parent != "" && *id != "") {
		flag.Usage()
		return
	}

	if os.Getenv("GDRIVE_TOKEN") == "" {
		fmt.Println("Go to", driveConfig.AuthCodeURL(""))
		code := ""
		fmt.Scanln(&code)
		token, err := driveConfig.Exchange(oauth2.NoContext, code)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println("Set GDRIVE_TOKEN to", token.RefreshToken)
		return
	}

	driveClient := driveConfig.Client(oauth2.NoContext, &oauth2.Token{
		RefreshToken: os.Getenv("GDRIVE_TOKEN"),
	})
	driveService, err := drive.New(driveClient)
	if err != nil {
		log.Panic(err)
	}

	targetID := *id
	if *parent != "" {
		f := &drive.File{
			Name: filepath.Base(*file),
			Parents: []string{
				*parent,
			},
			MimeType: *mime,
		}
		f, err = driveService.Files.Create(f).Do()
		if err != nil {
			log.Panic(err)
		}
		targetID = f.Id
	}
	input, err := os.Open(*file)
	if err != nil {
		log.Panic(err)
	}
	defer input.Close()
	_, err = driveService.Files.Update(targetID, &drive.File{
		MimeType: *mime,
	}).Media(input).Do()
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("https://drive.google.com/open?id=%s\n", targetID)
}
