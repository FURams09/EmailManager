package main

import (
				"bufio"
        "encoding/json"
        "fmt"
        "io/ioutil"
        "log"
        "net/http"
				"os"
				"sort"
				"strings"

        "golang.org/x/net/context"
        "golang.org/x/oauth2"
        "golang.org/x/oauth2/google"
        "google.golang.org/api/gmail/v1"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
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
type message struct {
	size    int64
	gmailID string
	date    string // retrieved from message header Date
	snippet string
	sender string // retrieved from message header From
}
func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
					log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
					log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.New(client)
	if err != nil {
					log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}
	var total int64
	msgs := []message{}

	user := "me"
	labelsToGet := []string {
		"FOOTBALL GUYS",
		"RECIPES",
	}

	for _, l := range labelsToGet {
		req := srv.Users.Messages.List(user)
		labelQuery := fmt.Sprintf("label:%v", l)
		req.Q(labelQuery)
		r, _ := req.Do()

		log.Printf("Processing %v messages for %v...\n", len(r.Messages), labelQuery)
		for _, m := range r.Messages {
			msg, err := srv.Users.Messages.Get("me", m.Id).Do()
			if err != nil {
				log.Fatalf("Unable to retrieve message %v: %v", m.Id, err)
			}
			
			total += msg.SizeEstimate
			date := ""
			sender := ""
			for _, h := range msg.Payload.Headers {
				switch h.Name {
				case "Date":
					date = h.Value
					break;
				case "From":
					sender = h.Value
					break;
				}
			
			}		
			msgs = append(msgs, message{
				size:    msg.SizeEstimate,
				gmailID: msg.Id,
				date:    date,
				snippet: msg.Snippet,
				sender: sender,
			})
		}		
		
	}

	log.Printf("total: %v\n", total)

	sortBySize(msgs)
	reader := bufio.NewReader(os.Stdin)
	count, deleted := 0, 0
	for _, m := range msgs {
		count++
		fmt.Printf("\nMessage URL: https://mail.google.com/mail/u/0/#all/%v\n", m.gmailID)
		fmt.Printf("Size: %v, Date: %v, Snippet: %q\n", m.size, m.date, m.snippet)
		fmt.Printf("Options: (d)elete, (s)kip, (q)uit: [s] ")
		val := ""
		if val, err = reader.ReadString('\n'); err != nil {
			log.Fatalf("unable to scan input: %v", err)
		}
		val = strings.TrimSpace(val)
		switch val {
		case "d": // delete message
			if err := srv.Users.Messages.Delete("me", m.gmailID).Do(); err != nil {
				log.Fatalf("unable to delete message %v: %v", m.gmailID, err)
			}
			log.Printf("Deleted message %v.\n", m.gmailID)
			deleted++
		case "q": // quit
			log.Printf("Done.  %v messages processed, %v deleted\n", count, deleted)
			os.Exit(0)
		default:
		}
	}
}

type messageSorter struct {
	msg  []message
	less func(i, j message) bool
}

func sortBySize(msg []message) {
	sort.Sort(messageSorter{msg, func(i, j message) bool {
		return i.size > j.size
	}})
}

func (s messageSorter) Len() int {
	return len(s.msg)
}

func (s messageSorter) Swap(i, j int) {
	s.msg[i], s.msg[j] = s.msg[j], s.msg[i]
}

func (s messageSorter) Less(i, j int) bool {
	return s.less(s.msg[i], s.msg[j])
}