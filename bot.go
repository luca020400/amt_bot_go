package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	port = "5555"

	apiVersion = "v1"
	url        = fmt.Sprintf("http://localhost:5555/api/%s", apiVersion)

	fns = template.FuncMap{
		"plus1": func(x int) int {
			return x + 1
		},
	}

	stopTemplate = `{{if .Name}}{{$len := len .Stops}}Fermata: {{.Name}}
{{if gt $len 0}}{{range $index, $stopData := .Stops}}
Numero Autobus: {{$stopData.Line}}
Direzione: {{$stopData.Dest}}
Orario di arrivo: {{$stopData.Time}}
Tempo rimanente: {{$stopData.ETA}}{{if lt (plus1 $index) $len}}
{{end}}
{{end}}{{else}}
Nessun autobus in arrivo{{end}}{{else}}Fermata non esistente{{end}}`
	lineTemplate = `{{$len := len .Lines}}{{if gt $len 0}}{{range $lineData := .Lines}}
{{$lineData.Direction}}
{{$len := len $lineData.Times}}{{range $index, $time := $lineData.Times}}{{$time}}{{if lt (plus1 $index) $len}}, {{end}}{{end}}
{{end}}{{else}}Linea non esistente{{end}}`
)

type StopData struct {
	Line string `json:"line"`
	Dest string `json:"dest"`
	Time string `json:"time"`
	ETA  string `json:"eta"`
}

type Stop struct {
	Name  string     `json:"name"`
	Stops []StopData `json:"stops"`
}

type LineData struct {
	Direction string   `json:"direction"`
	Times     []string `json:"times"`
}

type Line struct {
	Lines []LineData `json:"lines"`
}

func execTempl(template *template.Template, data interface{}) (string, error) {
	buf := &bytes.Buffer{}
	err := template.Execute(buf, data)
	return buf.String(), err
}

func downloadStop(code string) Stop {
	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", url, "stop", code))
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var stop Stop
	err = json.Unmarshal(body, &stop)
	if err != nil {
		log.Panic(err)
	}

	return stop
}

func beautifyStop(stop Stop) string {
	template := template.Must(template.New("stop").Funcs(fns).Parse(stopTemplate))
	message, err := execTempl(template, stop)
	if err != nil {
		log.Panic(err)
	}

	return message
}

func getStopMessage(code string) string {
	stop := downloadStop(code)
	message := beautifyStop(stop)
	return message
}

func downloadLine(code string) Line {
	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", url, "line", code))
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var line Line
	err = json.Unmarshal(body, &line)
	if err != nil {
		log.Panic(err)
	}

	return line
}

func beautifyLine(line Line) string {
	template := template.Must(template.New("line").Funcs(fns).Parse(lineTemplate))
	message, err := execTempl(template, line)
	if err != nil {
		log.Panic(err)
	}

	return message
}

func getLineMessage(code string) string {
	line := downloadLine(code)
	message := beautifyLine(line)
	return message
}

func getKey() string {
	dat, err := ioutil.ReadFile("key.txt")
	if err != nil {
		log.Panic(err)
	}

	return strings.Trim(string(dat), "\n")
}

func main() {
	bot, err := tgbotapi.NewBotAPI(getKey())
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		var message string
		stopRegex, _ := regexp.Compile("\\d{4}")
		lineRegex, _ := regexp.Compile("[A-Z0-9]{1,3}")

		if stopRegex.MatchString(update.Message.Text) {
			message = getStopMessage(update.Message.Text)
		} else if lineRegex.MatchString(update.Message.Text) {
			message = getLineMessage(update.Message.Text)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)

		bot.Send(msg)
	}
}
