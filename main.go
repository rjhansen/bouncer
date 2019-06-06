package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/user"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Configuration struct {
	Host string `json:"host"`
	Port int `json:"port"`
	Login string `json:"login"`
	Password string `json:"password"`
	Wiki string `json:"wiki_base"`
	ActiveCharacterPage string `json:"active_character_page"`
	ActiveCharacterRegexp string `json:"active_character_regex"`
	OnMushAs string `json:"on_mush_as_regex"`
	FingerRegexp string `json:"finger_regex"`
	RecentLoginRegexp string `json:"recent_login_regex"`
	Connect string `json:"on_connect"`
	Disconnect string `json:"on_disconnect"`
	FingerCommand string `json:"finger_command"`
}

type ProcessedConfiguration struct {
	Host string
	Port int
	Login string
	Password string
	Wiki string
	ActiveCharacterPage string
	ActiveCharacterRegexp *regexp.Regexp
	OnMushAs string
	FingerRegexp *regexp.Regexp
	RecentLoginRegexp *regexp.Regexp
	Connect string
	Disconnect string
	FingerCommand string

}

var config ProcessedConfiguration

func FetchURL(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	rawData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	err = resp.Body.Close()
	if err != nil {
		panic(err)
	}
	return string(rawData)
}

func GetListOfWikiPages() []string {
	rv := make([]string, 0)
	index := config.Wiki + config.ActiveCharacterPage
	rx := config.ActiveCharacterRegexp
	indexPage := FetchURL(index)
	for _, href := range rx.FindAllStringSubmatch(indexPage, -1) {
		rv = append(rv, config.Wiki + href[1])
	}
	return rv
}

func GetActiveCharacters() map[string]string {
	stop := make(chan bool)
	data := make(chan string)
	rv := make(map[string]string)
	// awaits name-url pairs (one string, tab-delimited) and builds up our
	// return value.  Since this is the only thing reading or writing to rv
	// during execution of GetActiveCharacters, we're assured of no race
	// conditions as we build rv.
	go func() {
		for {
			select {
			case line := <-data:
				elems := strings.Split(line, "\t")
				rv[elems[0]] = elems[1]
			case <-stop:
				return
			}
		}
	}()

	wg := sync.WaitGroup{}
	rx := regexp.MustCompile(config.OnMushAs)
	counter := 0

	for _, hyperlink := range GetListOfWikiPages() {
		// Fetch ten urls at a time for efficiency reasons.
		wg.Add(1)

		go func(url string) {
			matches := rx.FindStringSubmatch(FetchURL(url))
			if len(matches) >= 2 {
				data<-matches[1] + "\t" + url
			}
			wg.Done()
		}(hyperlink)

		counter += 1
		if counter == 10 {
			wg.Wait()
			counter = 0
		}
	}
	if counter > 0 {
		wg.Wait()
	}

	// signals our rv-building gofunc it's time to stop
	stop<-true

	return rv
}

func GetMUSHData(names []string) string {
	var rv string
	stop := make(chan bool)
	host := config.Host + ":" + strconv.Itoa(config.Port)
	conn, err := net.Dial("tcp", host)
	if err != nil {
		panic(err)
	}
	defer func() {
		_, _ = conn.Write([]byte(config.Disconnect + "\r\n"))
		_, _ = conn.Write([]byte("QUIT\r\n"))
		_ = conn.Close()
	}()


	// Spins up and does nothing but accumulate data from the MUSH until
	// such time as it's signaled to stop.
	go func() {
		buf := make([]byte, 65536)
		for {
			select {
			case <-stop:
				stop<-false
				return
			default:
				timeout := time.Now().Add(100 * time.Millisecond)
				err := conn.SetReadDeadline(timeout)
				if err != nil {
					panic(err)
				}
				bytesRead, err := conn.Read(buf)
				if err != nil {
					if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
						continue
					} else {
						panic(err)
					}
				}
				rv += string(buf[:bytesRead])
			}
		}
	}()

	_, err = conn.Write([]byte("connect " + config.Login + " " +
		config.Password + "\r\n"))
	if err != nil {
		panic(err)
	}

	time.Sleep(1000 * time.Millisecond)

	_, err = conn.Write([]byte(config.Connect + "\r\n"))
	if err != nil {
		panic(err)
	}

	for _, name := range names {
		_, err = conn.Write([]byte(config.FingerCommand + " " + name + "\r\n"))
		time.Sleep(100 * time.Millisecond)
	}
	stop<-true
	<-stop
	return rv
}

func LoadConfig() {
	var tmpConfig Configuration
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	filename := path.Join(usr.HomeDir, ".bouncer.json")
	jsonFile, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	buffer, err := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(buffer, &tmpConfig)
	if err != nil {
		panic(err)
	}
	err = jsonFile.Close()
	if err != nil {
		panic(err)
	}

	config.Host = strings.TrimSpace(tmpConfig.Host)
	config.Port = tmpConfig.Port
	config.Login = strings.TrimSpace(tmpConfig.Login)
	config.Password = strings.TrimSpace(tmpConfig.Password)
	config.Wiki = strings.TrimSpace(tmpConfig.Wiki)
	config.ActiveCharacterPage = strings.TrimSpace(tmpConfig.ActiveCharacterPage)
	config.ActiveCharacterRegexp = regexp.MustCompile(tmpConfig.ActiveCharacterRegexp)
	config.OnMushAs = strings.TrimSpace(tmpConfig.OnMushAs)
	config.FingerRegexp = regexp.MustCompile(tmpConfig.FingerRegexp)
	config.RecentLoginRegexp = regexp.MustCompile(tmpConfig.RecentLoginRegexp)
	config.Connect = strings.TrimSpace(tmpConfig.Connect)
	config.Disconnect = strings.TrimSpace(tmpConfig.Disconnect)
	config.FingerCommand = strings.TrimSpace(tmpConfig.FingerCommand)


	if config.Port < 1024 || config.Port > 65535 {
		panic(errors.New("Invalid port number"))
	}
}

func main() {
	LoadConfig()

	charmap := GetActiveCharacters()

	mapkeys := reflect.ValueOf(charmap).MapKeys()
	names := make([]string, len(mapkeys))
	for index, value := range mapkeys {
		names[index] = value.String()
	}

	data := GetMUSHData(names)
	rows := strings.Split(data, "\n")

	for index, row := range rows {
		row = strings.TrimSpace(row)
		match := config.FingerRegexp.FindStringSubmatch(row)
		if len(match) > 0 && index < (len(rows)-1) {
			nextrow := rows[index+1]
			if config.RecentLoginRegexp.MatchString(nextrow) {
				delete(charmap, match[1])
			}
		}
	}
	fmt.Print("+request Wiki cleanup=The following characters need to be reviewed for activity:%r%r")
	for k, v := range charmap {
		foo := " "
		if len(k) < 10 {
			foo = "[space(" + strconv.Itoa(10-len(k)) + ")]"
		}
		fmt.Print("%t" + k + foo + v + " %r")
	}
}
