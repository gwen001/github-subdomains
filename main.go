package main

import (
	"os"
	"fmt"
	"time"
	"flag"
	"sync"
	"bufio"
	"regexp"
	"strings"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"github.com/logrusorgru/aurora"
	tld "github.com/jpillora/go-tld"
)

type Search struct {
	keyword string
	sort string
	order string
	language string
	noise []string
	TotalCount int
}

type Config struct {
	domain string
	output string
	fpOutput *os.File
	token []string
	extend bool
	raw bool
	search string
	delay time.Duration
	DomainRegexp *regexp.Regexp
}

type item struct {
	HtmlUrl string `json:"html_url"`
}

type response struct {
	Message string `json:"message"`
	DocumentationUrl string `json:"documentation_url"`
	TotalCount int `json:"total_count"`
	Items []item `json:"items"`
}

var au = aurora.NewAurora(true)
var config = Config{}
var t_history_urls []string
var t_subdomains []string
var t_search []Search
var t_languages []string
var t_noise []string


func readTokenFromFile() string {

	fp, err := os.Open(".tokens")
    defer fp.Close()

    if err != nil {
        return ""
    }

	var line string
	var token []string
    var reader = bufio.NewReader(fp)

    for {
		line, err = reader.ReadString('\n')

        if err != nil {
            break
		}

		token = append(token, line)
    }

	return strings.Join(token, ",")
}


func loadLanguages() bool {

	fp, err := os.Open("languages.txt")
    defer fp.Close()

    if err != nil {
        return false
    }

	var line string
    var reader = bufio.NewReader(fp)

    for {
		line, err = reader.ReadString('\n')

        if err != nil {
            break
		}

		line = strings.TrimSpace( line )
		if len(line) > 0 {
			t_languages = append(t_languages, line)
		}
	}

	return true
}


func loadNoise() bool {

	fp, err := os.Open("noise.txt")
    defer fp.Close()

    if err != nil {
        return false
    }

	var line string
    var reader = bufio.NewReader(fp)

    for {
		line, err = reader.ReadString('\n')

        if err != nil {
            break
		}

		line = strings.TrimSpace( line )
		if len(line) > 0 {
			t_noise = append(t_noise, line)
		}
	}

	return true
}


func githubSearch(token string, current_search Search, page int) response {

	var search = current_search.keyword

	if len(current_search.language) > 0 {
		search = fmt.Sprintf("%s+language:%s", search, current_search.language)
	}

	if len(current_search.noise) > 0 {
		search = fmt.Sprintf("%s+language:%s", search, current_search.noise)
	}

	var url = fmt.Sprintf("https://api.github.com/search/code?per_page=100&sort=%s&order=%s&q=%s&page=%d", current_search.sort, current_search.order, search, page )
	PrintInfos( "debug", url )

	client := http.Client{ Timeout: time.Second * 5 }

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Set("Authorization", "token "+token)

	res, getErr := client.Do(req)
	if getErr != nil {
		fmt.Println(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		fmt.Println(readErr)
	}

	r := response{}
	jsonErr := json.Unmarshal(body, &r)
	if jsonErr != nil {
		fmt.Println(jsonErr)
	}

	return r
}


func getCode( i item ) string {
	var raw_url = getRawUrl(i.HtmlUrl)
	// PrintInfos("debug", raw_url)

	client := http.Client{ Timeout: time.Second * 5 }

	req, err := http.NewRequest("GET", raw_url, nil)
	if err != nil {
		fmt.Println(err)
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		fmt.Println(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		fmt.Println(readErr)
	}

	return string(body)
}


func doItem(i item) {

	var t_subs [][]byte

	if inArray(i.HtmlUrl,t_history_urls) {
		// PrintInfos( "debug", fmt.Sprintf("url already checked: %s",i.HtmlUrl) )
	} else {

		// PrintInfos( "debug", i.HtmlUrl )
		t_history_urls = append(t_history_urls, i.HtmlUrl)

		var code = getCode( i )
		t_subs = extractSubdomains( code, config.DomainRegexp )

		if len(t_subs) > 0 {
			var print_url = false
			for _, sub := range t_subs {
				var str_sub = string(sub)
				if !inArray(str_sub,t_subdomains) {
					t_subdomains = append( t_subdomains, str_sub )
					if !print_url {
						print_url = true
						PrintInfos( "info", i.HtmlUrl )
					}
					PrintInfos( "found", str_sub )
					config.fpOutput.WriteString(str_sub+"\n")
					config.fpOutput.Sync()
				}
			}
		}
	}
}


func main() {

	var token string

	flag.StringVar( &config.domain, "d", "", "domain you are looking for (required)" )
	flag.BoolVar( &config.extend, "e", false, "extended mode, also look for <dummy>example.com" )
	flag.BoolVar( &config.raw, "raw", false, "raw output" )
	flag.StringVar( &token, "t", "", "github token (required)" )
	flag.StringVar( &config.output, "o", "", "output file, default: <domain>.txt" )
	flag.Parse()

	if config.domain == "" {
		flag.Usage()
		fmt.Printf("\ndomain not found\n")
		os.Exit(-1)
	}

	if config.output == "" {
		dir, _ := os.Getwd()
		config.output = dir + "/" + config.domain + ".txt"
	}

	fp, outErr := os.Create( config.output )
	if outErr != nil {
		fmt.Println(outErr)
		os.Exit(-1)
	}

	config.fpOutput = fp
	// defer fp.Close()

	u, _ := tld.Parse("http://"+config.domain)

	if config.extend {
		config.search = u.Domain
		config.DomainRegexp = regexp.MustCompile( `[0-9a-z_\-\.]+\.` + u.Domain )
		// domain_regexp = r'([0-9a-z_\-\.]+\.([0-9a-z_\-]+)?'+t_host_parse.domain+'([0-9a-z_\-\.]+)?\.[a-z]{1,5})'
	} else {
		config.search = u.Domain + "." + u.TLD
		config.DomainRegexp = regexp.MustCompile( `[0-9a-z_\-\.]+\.` + u.Domain + "\\." + u.TLD )
		// domain_regexp = r'(([0-9a-z_\-\.]+)\.' + _domain.replace('.','\.')+')'
	}

	config.search = "%22" + strings.ReplaceAll(config.search, "-", "%2D") + "%22"

	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
		if token == "" {
			token = readTokenFromFile()
			if token == "" {
				flag.Usage()
				fmt.Printf("\ntoken not found\n")
				os.Exit(-1)
			}
		}
	}

	config.token = strings.Split(token, ",")

	if !config.raw {
		banner()
	}

	var page int
	var current_token = 0
	var n_token = len(config.token)
	var r response
	var wg sync.WaitGroup
	var max_procs = make(chan bool, 30)

	config.delay = time.Duration( 60.0 / (30*float64(n_token)) * 1000 + 200)
	// fmt.Printf("%.3f",config.delay)

	loadLanguages()
	loadNoise()
	displayConfig()

	t_search = append( t_search, Search{keyword:config.search, sort:"indexed", order:"desc"} )
	// t_search = append( t_search, Search{sort:"indexed", order:"asc"} )
	// t_search = append( t_search, Search{sort:"", order:"desc"} )
	var n_search = len(t_search)
	var index = 0
	var current_search Search

	for index < n_search {
	// for k,s := range t_search {

		current_search = t_search[index]
		// PrintInfos( "debug", fmt.Sprintf("sort:%s, order:%s, language:%s", s.sort, s.order, s.language) )
		PrintInfos( "debug", fmt.Sprintf("sort:%s, order:%s, language:%s, noise:%s", current_search.sort, current_search.order, current_search.language, current_search.noise) )

		page = 1

		for {

			time.Sleep( config.delay * time.Millisecond )

			var ct = current_token%n_token
			r = githubSearch( config.token[ct], current_search, page )
			// r = githubSearch( config.token[ct], config.search, page, s.sort, s.order, s.language )
			current_token++

			if len(r.Message) > 0 {
				// fmt.Println(r.Message)
				// fmt.Println(r.DocumentationUrl)
				if strings.HasPrefix(r.Message,"Only the first") {
					// Only the first 1000 search results are available
					PrintInfos("debug", "search limit reached")
					break
				} else if strings.HasPrefix(r.Message,"You have triggered an abuse detection mechanism") {
					// You have triggered an abuse detection mechanism. Please wait a few minutes before you try again.
					PrintInfos("debug", "token limit reached, token removed from the list")
					config.token = reslice( config.token, ct )
					n_token--
					if n_token == 0 {
						PrintInfos("error", "tokens limit reached, no more token available, exiting...")
						os.Exit(-1)
					}
					continue
				}
			}

			// fmt.Println(len(t_search))
			if page == 1 {
				t_search[index].TotalCount = r.TotalCount

				if r.TotalCount > 1000 {
					if current_search.language == "" {
						addSearchLanguage( current_search )
						PrintInfos( "debug", "current search returned too much results, language filter added for later search" )
					} else {
						addSearchNoise( current_search )
						PrintInfos( "debug", "current search returned too much results, noise added for later search" )
					}
					n_search = len(t_search)
				}
			}
			// fmt.Println(len(t_search))

			for _, i := range r.Items {
				wg.Add(1)
				go func(i item) {
					defer wg.Done()
					max_procs<-true
					doItem( i )
					<-max_procs
				}(i)
			}
			wg.Wait()

			page++
			// break
		}

		index++
	}

	PrintInfos( "", fmt.Sprintf("%d searches performed",n_search) )
	PrintInfos( "", fmt.Sprintf("%d subdomains found",len(t_subdomains)) )
}


func addSearchLanguage( current_search Search ) {

	for _,language := range t_languages {
		var new_search Search
		new_search.sort = current_search.sort
		new_search.order = current_search.order
		new_search.language = language
		t_search = append( t_search, new_search )
	}
}


func addSearchNoise( current_search Search ) {

	for _,noise := range t_noise {
		if !inArray(noise,current_search.noise) {
			var new_search Search
			new_search.sort = current_search.sort
			new_search.order = current_search.order
			new_search.language = current_search.language
			new_search.noise = append( current_search.noise, noise )
			t_search = append( t_search, new_search )
		}
	}
}


func inArray(str string, array []string) bool {
	for _,i := range array {
		if i == str {
			return true
		}
	}
	return false
}


func extractSubdomains(code string, domain_regexp *regexp.Regexp ) [][]byte {
	return domain_regexp.FindAll([]byte(code), -1)
}


func getRawUrl( html_url string ) string {
    var raw_url = html_url
    raw_url = strings.Replace( raw_url, "https://github.com/", "https://raw.githubusercontent.com/", -1 )
    raw_url = strings.Replace( raw_url, "/blob/", "/", -1 )
	return raw_url
}


func reslice(s []string, index int) []string {
    return append(s[:index], s[index+1:]...)
}


func displayConfig() {
	PrintInfos( "", fmt.Sprintf("Domain:%s, Output:%s, Tokens:%d, Delay:%.0fms",config.domain,config.output,len(config.token),float32(config.delay)) )
	PrintInfos( "", fmt.Sprintf("Languages:%d, Noise:%d",len(t_languages),len(t_noise)) )
}


func PrintInfos(infos_type string, str string) {

	if config.raw && infos_type == "found" {
		fmt.Println( str )
	} else if !config.raw {
		str = fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), str )

		switch infos_type {
			case "debug":
				fmt.Println( au.Gray(13,str).Bold() )
			case "info":
				fmt.Println( au.Yellow(str).Bold() )
			case "found":
				fmt.Println( au.Green(str).Bold() )
			case "error":
				fmt.Println( au.Red(str).Bold() )
			default:
				fmt.Println( au.White(str).Bold() )
		}
	}
}


func banner() {
	fmt.Print("\n")
	fmt.Print(au.BrightMagenta(`        █▀▀`))
	fmt.Print(au.BrightWhite(` ▀█▀ ▀█▀ █ █ █ █ █▀▄   `))
	fmt.Print(au.BrightMagenta(`█▀▀`))
	fmt.Println(au.BrightWhite(` █ █ █▀▄ █▀▄ █▀█ █▄█ █▀█ ▀█▀ █▀█ █▀▀`))
	fmt.Print(au.BrightMagenta(`        █ █`))
	fmt.Print(au.BrightWhite(`  █   █  █▀█ █ █ █▀▄   `))
	fmt.Print(au.BrightMagenta(`▀▀█`))
	fmt.Println(au.BrightWhite(` █ █ █▀▄ █ █ █ █ █ █ █▀█  █  █ █ ▀▀█`))
	fmt.Print(au.BrightMagenta(`        ▀▀▀`))
	fmt.Print(au.BrightWhite(` ▀▀▀  ▀  ▀ ▀ ▀▀▀ ▀▀    `))
	fmt.Print(au.BrightMagenta(`▀▀▀`))
	fmt.Print(au.BrightWhite(` ▀▀▀ ▀▀  ▀▀  ▀▀▀ ▀ ▀ ▀ ▀ ▀▀▀ ▀ ▀ ▀▀▀
	`))
	fmt.Print("                    by @gwendallecoguic                          \n\n")
}
