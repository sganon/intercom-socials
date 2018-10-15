package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx"
	"github.com/urfave/cli"
)

type User struct {
	Email          string `json:"email"`
	SocialProfiles struct {
		Type    string `json:"type"`
		Socials []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"social_profiles"`
	} `json:"social_profiles"`
}

type UsersResponse struct {
	Type  string `json:"type"`
	Pages struct {
		Type string  `json:"type"`
		Next *string `json:"next"`
		Page int     `json:"page"`
	} `json:"pages"`
	Users []User `json:"users"`
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "apiToken, T",
			Value:  "",
			EnvVar: "INTERCOM_API_TOKEN",
		},
		cli.StringFlag{
			Name:   "logLevel, L",
			Value:  "info",
			EnvVar: "INTERCOM_LOG_LEVEL",
		},
		cli.StringFlag{
			Name:  "output, O",
			Value: "IntercomSocials.xlsx",
		},
		cli.BoolFlag{
			Name: "ignoreEmpty, I",
		},
	}
	app.Action = func(c *cli.Context) (err error) {
		setLogLevel(c.String("logLevel"))
		var users []User
		client := http.DefaultClient
		url := "https://api.intercom.io/users?page=1"
		for {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("[INTERCOM ERROR]: error making request:\n\t%s", err)
			}
			req.Header.Add("Authorization", "Bearer "+c.String("apiToken"))
			req.Header.Add("Accept", "application/json")
			res, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("[INTERCOM ERROR]: error sending request:\n\t%s", err)
			}
			defer res.Body.Close()
			decoder := json.NewDecoder(res.Body)
			var body UsersResponse
			err = decoder.Decode(&body)
			if err != nil {
				return fmt.Errorf("[INTERCOM ERROR]: error decoding response body:\n\t%s", err)
			}
			log.Debugf("page n° %d fetched", body.Pages.Page)
			if body.Pages.Next == nil {
				break
			}
			url = *body.Pages.Next
			users = append(users, body.Users...)
		}
		networks := make(map[string]interface{})
		for _, u := range users {
			for _, n := range u.SocialProfiles.Socials {
				if _, prs := networks[n.Name]; !prs {
					networks[n.Name] = true
				}
			}
		}
		file := xlsx.NewFile()
		sheet, err := file.AddSheet("Sheet1")
		if err != nil {
			return fmt.Errorf("[INTERCOM ERROR]: error creating sheet:\n\t%s", err)
		}
		row := sheet.AddRow()
		cell := row.AddCell()
		i := 0
		indexes := make(map[string]int)
		cell.Value = "Email"
		indexes["Email"] = i
		for n := range networks {
			if _, prs := indexes[strings.Title(n)]; !prs {
				i++
				cell = row.AddCell()
				cell.Value = strings.Title(n)
				indexes[strings.Title(n)] = i
			}
		}
		for _, u := range users {
			if len(u.SocialProfiles.Socials) > 0 || c.Bool("ignoreEmpty") {
				row = sheet.AddRow()
				cell = row.AddCell()
				cell.Value = u.Email
				for _, s := range u.SocialProfiles.Socials {
					sI := indexes[strings.Title(s.Name)]
					if len(row.Cells) < sI {
						l := len(row.Cells)
						for l <= sI {
							row.AddCell()
							l++
						}
						row.Cells[sI].Value = s.URL
					}
				}
			}
		}
		err = file.Save(c.String("output"))
		if err != nil {
			return fmt.Errorf("[INTERCOM ERROR]: error saving file:\n\t%s", err)
		}
		log.Debugf("File %s written", c.String("output"))
		log.Info("Done")
		return err
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}

func setLogLevel(level string) {
	switch level {
	case "debug":
		log.SetLevel(log.DebugLevel)
		log.Debugln("Debug Level")
	case "info":
		log.SetLevel(log.InfoLevel)
		log.Infoln("Info level")
	case "warn":
		log.SetLevel(log.WarnLevel)
		log.Warningln("Warning mode")
	case "error":
		log.SetLevel(log.ErrorLevel)
		log.Errorln("Error level")
	default:
		log.Fatalln("Unrecognised log level")
	}
}
