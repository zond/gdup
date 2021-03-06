package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mitchellh/ioprogress"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
)

func main() {
	file := flag.String("file", "", "File to upload.")
	parent := flag.String("parent", "", "Google Drive ID of parent directory. Provide either -parent or -id.")
	id := flag.String("id", "", "ID of file to upload to. Provide either -parent or -id.")
	mime := flag.String("mime", "", "MIME type of uploaded file.")
	verbose := flag.Bool("verbose", false, "Whether to be verbose with the progress.")
	quiet := flag.Bool("quiet", false, "Whether to supress the final output of the link to the file on Google Drive.")
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

	input, err := os.Open(*file)
	if err != nil {
		log.Panic(err)
	}
	defer input.Close()

	upload := func(id string) error {
		updateFile := &drive.File{}
		if *mime != "" {
			updateFile.MimeType = *mime
		}
		stat, err := input.Stat()
		if err != nil {
			return err
		}
		var realInput io.Reader = input
		if *verbose {
			realInput = &ioprogress.Reader{
				Reader:   input,
				Size:     stat.Size(),
				DrawFunc: ioprogress.DrawTerminalf(os.Stderr, ioprogress.DrawTextFormatBar(80)),
			}
		}
		_, err = driveService.Files.Update(id, updateFile).Media(realInput).Do()
		return err
	}

	targetID := *id
	if *parent != "" {
		f := &drive.File{
			Name: filepath.Base(*file),
			Parents: []string{
				*parent,
			},
		}
		if *mime != "" {
			f.MimeType = *mime
		}
		f, err = driveService.Files.Create(f).Do()
		if err != nil {
			log.Panic(err)
		}
		targetID = f.Id
	} else if *id != "" {
		sum := md5.New()
		io.Copy(sum, input)
		localMd5 := hex.EncodeToString(sum.Sum(nil))
		if _, err := input.Seek(0, 0); err != nil {
			log.Panic(err)
		}
		f, err := driveService.Files.Get(targetID).Fields("md5Checksum").Do()
		if err != nil {
			log.Panic(err)
		}
		if localMd5 == f.Md5Checksum {
			if *verbose {
				fmt.Fprintf(os.Stderr, "%#v == %#v\n", localMd5, f.Md5Checksum)
			}
			upload = func(id string) error { return nil }
		}
	}
	if err := upload(targetID); err != nil {
		log.Panic(err)
	}
	if !*quiet {
		fmt.Printf("https://drive.google.com/open?id=%s\n", targetID)
	}
}
