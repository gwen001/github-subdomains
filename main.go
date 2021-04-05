package main

import (
	"os"
	"fmt"
	"math"
	"sort"
	"time"
	"flag"
	"sync"
	"regexp"
	"strings"
	"net/url"
	"net/http"
	"io/ioutil"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/logrusorgru/aurora"
	tld "github.com/jpillora/go-tld"
)

type Token struct {
	datoken string
	disabled_ts int64
}

type Search struct {
	signature string
	keyword string
	sort string
	order string
	language string
	noise []string
	TotalCount int
}

type Config struct {
	stop_notoken bool
	quick_mode bool
	domain string
	output string
	fpOutput *os.File
	tokens []Token
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
var t_languages = []string{"JavaScript","Python","Java","Go","Ruby","PHP","Shell","CSV","Markdown","XML","JSON","Text","CSS","HTML","Perl","ActionScript","Lua","C","C%2B%2B","C%23"}
var t_noise = []string{"api","private","secret","internal","corp","development","production"}


func parseToken( token string ) {

	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
		if token == "" {
			token = readTokenFromFile(".tokens")
			if token == "" {
				flag.Usage()
				fmt.Printf("\ntoken not found\n")
				os.Exit(-1)
			}
		}
	} else {
		if _, err := os.Stat(token); os.IsNotExist(err) {
			// path/to/whatever does not exist
		} else {
			token = readTokenFromFile( token )
		}
	}

	var t_tokens = strings.Split(token, ",")
	var re = regexp.MustCompile(`[0-9a-f]{40}|ghp_[a-zA-Z0-9]{36}`)

	for _,t := range t_tokens {
		if re.MatchString(t) {
			config.tokens = append( config.tokens, Token{datoken:t,disabled_ts:0} )
		}
	}
}

func readTokenFromFile( tokenfile string ) string {

	b, err := ioutil.ReadFile( tokenfile )

    if err != nil {
        return ""
    }

	var t_token []string

	for _,l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace( l )
		if len(l) > 0 && !inArray(l,t_token) {
			t_token = append(t_token, l)
		}
	}

	return strings.Join(t_token, ",")
}


func loadLanguages(filename string) bool {

	t_languages = nil

	if filename == "none" {
		return true
	}

	b, err := ioutil.ReadFile(filename)

    if err != nil {
		PrintInfos( "error", fmt.Sprintf("can't open language file: %s",filename) )
        os.Exit(-1)
    }

	for _,l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace( l )
		if len(l) > 0 && !inArray(l,t_languages) {
			t_languages = append(t_languages, l)
		}
	}

	return true
}


func loadNoise(filename string) bool {

	t_noise = nil

	if filename == "none" {
		return true
	}

	b, err := ioutil.ReadFile(filename)

    if err != nil {
		PrintInfos( "error", fmt.Sprintf("can't open noise file: %s",filename) )
        os.Exit(-1)
    }

	for _,l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace( l )
		if len(l) > 0 && !inArray(l,t_noise) {
			t_noise = append(t_noise, l)
		}
	}

	return true
}


func githubSearch(token string, current_search Search, page int) response {

	defer func() {
        if r := recover(); r != nil {
            // fmt.Println("Recovered in f", r)
        }
    }()

	var search = current_search.keyword

	if len(current_search.language) > 0 {
		search = fmt.Sprintf("%s+language:%s", search, current_search.language)
	}

	if len(current_search.noise) > 0 {
		search = fmt.Sprintf("%s+%s", search, strings.Join(current_search.noise,"+"))
	}

	// var url = fmt.Sprintf("https://api.github.com/search/code?per_page=100&sort=%s&order=%s&q=%s&page=%d", current_search.sort, current_search.order, search, page )
	var url = fmt.Sprintf("https://api.github.com/search/code?per_page=100&s=%s&type=Code&o=%s&q=%s&page=%d", current_search.sort, current_search.order, search, page )
	PrintInfos( "debug", url )

	client := http.Client{ Timeout: time.Second * 5 }

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		PrintInfos( "error", fmt.Sprintf("%s",err) )
	}

	req.Header.Set("Authorization", "token "+token)

	res, getErr := client.Do(req)
	if getErr != nil {
		PrintInfos( "error", fmt.Sprintf("%s",getErr) )
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		PrintInfos( "error", fmt.Sprintf("%s",readErr) )
	}

	r := response{}
	jsonErr := json.Unmarshal(body, &r)
	if jsonErr != nil {
		PrintInfos( "error", fmt.Sprintf("%s",jsonErr) )
	}

	return r
}


func getCode( i item ) string {

	defer func() {
        if r := recover(); r != nil {
            // fmt.Println("Recovered in f", r)
        }
    }()

	var raw_url = getRawUrl(i.HtmlUrl)

	client := http.Client{ Timeout: time.Second * 5 }

	req, err := http.NewRequest("GET", raw_url, nil)
	if err != nil {
		PrintInfos( "error", fmt.Sprintf("%s",err) )
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		PrintInfos( "error", fmt.Sprintf("%s",getErr) )
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		PrintInfos( "error", fmt.Sprintf("%s",readErr) )
	}

	return string(body)
}


func cleanSubdomain(sub []byte) string {
	var clean_sub = string(sub)
	clean_sub = strings.ToLower( clean_sub )
	clean_sub = strings.TrimLeft( clean_sub, "." )
	if strings.Index(clean_sub,"2f") == 0 {
		clean_sub = clean_sub[2:]
	}
	if strings.Index(clean_sub,"252f") == 0 {
		clean_sub = clean_sub[4:]
	}
	var re = regexp.MustCompile( `^u00[0-9a-f][0-9a-f]` )
	clean_sub = re.ReplaceAllString( clean_sub, "" )

	return clean_sub
}


func doItem(i item) {

	var t_match [][]byte

	if inArray(i.HtmlUrl,t_history_urls) {
		// PrintInfos( "debug", fmt.Sprintf("url already checked: %s",i.HtmlUrl) )
	} else {

		t_history_urls = append(t_history_urls, i.HtmlUrl)

		var code = getCode( i )
		t_match = performRegexp( code, config.DomainRegexp )

		if len(t_match) > 0 {
			var print_url = false
			for _, match := range t_match {
				var str_match = cleanSubdomain( match )
				if !inArray(str_match,t_subdomains) {
					t_subdomains = append( t_subdomains, str_match )
					if !print_url {
						print_url = true
						PrintInfos( "info", i.HtmlUrl )
					}
					PrintInfos( "found", str_match )
					config.fpOutput.WriteString(str_match+"\n")
					config.fpOutput.Sync()
				}
			}
		}
	}
}


func getNextToken( token_index int, n_token int ) int {

	token_index = (token_index+1) % n_token

	for k:=token_index ; k<n_token ; k++ {
		if config.tokens[k].disabled_ts == 0 || config.tokens[k].disabled_ts < time.Now().Unix() {
			config.tokens[k].disabled_ts = 0
			return k
		}
	}

	return -1
}


func main() {

	var token string
	var f_language string
	var f_noise string

	flag.BoolVar( &config.quick_mode, "q", false, "quick mode, avoid extra searches with languages and noise added" )
	flag.StringVar( &config.domain, "d", "", "domain you are looking for (required)" )
	flag.BoolVar( &config.extend, "e", false, "extended mode, also look for <dummy>example.com" )
	flag.BoolVar( &config.raw, "raw", false, "raw output" )
	flag.StringVar( &token, "t", "", "github token (required), can be:\n  • a single token\n  • a list of tokens separated by comma\n  • a file containing 1 token per line\nif the options is not provided, the environment variable GITHUB_TOKEN is readed, it can be:\n  • a single token\n  • a list of tokens separated by comma" )
	flag.StringVar( &config.output, "o", "", "output file, default: <domain>.txt" )
	flag.BoolVar( &config.stop_notoken, "k", false, "exit the program when all tokens have been disabled" )
	// flag.StringVar( &f_language, "l", "", "language file (optional)" )
	// flag.StringVar( &f_noise, "n", "", "noise file (optional)" )
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
		config.DomainRegexp = regexp.MustCompile( `(?i)[0-9a-z\-\.]+\.([0-9a-z\-]+)?`+u.Domain+`([0-9a-z\-\.]+)?\.[a-z]{1,5}`)
	} else {
		config.search = u.Domain + "." + u.TLD
		config.DomainRegexp = regexp.MustCompile( `(?i)[0-9a-z\-\.]+\.` + u.Domain + "\\." + u.TLD )
	}

	config.search = "%22" + strings.ReplaceAll(url.QueryEscape(config.search), "-", "%2D") + "%22"

	parseToken( token )

	if !config.raw {
		banner()
	}

	var n_token = len(config.tokens)
	if n_token == 0 {
		flag.Usage()
		PrintInfos( "error", "token not found" )
		os.Exit(-1)
	}

	var wg sync.WaitGroup
	var max_procs = make(chan bool, 30)

	config.delay = time.Duration( 60.0 / (30*float64(n_token)) * 1000 + 200)

	if( config.quick_mode ) {
		t_languages = nil
		t_noise = nil
	} else {
		if f_language != "" {
			loadLanguages( f_language )
		}
		if f_noise != "" {
			loadNoise( f_noise )
		}
	}

	displayConfig()

	t_search = append( t_search, Search{keyword:config.search, sort:"indexed", order:"desc"} )

	var n_search = len(t_search)
	var search_index = 0
	var token_index = -1
	var current_search Search

	for search_index < n_search {

		current_search = t_search[search_index]
		PrintInfos( "debug", fmt.Sprintf("keyword:%s, sort:%s, order:%s, language:%s, noise:%s", current_search.keyword, current_search.sort, current_search.order, current_search.language, current_search.noise) )

		var max_page = 1

		for page:=1; page<=max_page; {

			time.Sleep( config.delay * time.Millisecond )

			// var ct = token_index%n_token
			token_index = getNextToken( token_index, n_token )

			if token_index < 0 {
				token_index = -1

				if( config.stop_notoken ) {
					PrintInfos("error", "no more token available, exiting")
					os.Exit(-1)
				}

				PrintInfos("error", "no more token available, waiting for another available token...")
				continue
			}

			var r = githubSearch( config.tokens[token_index].datoken, current_search, page )

			if len(r.Message) > 0 {
				// fmt.Println(r.Message)
				// fmt.Println(r.DocumentationUrl)
				if strings.HasPrefix(r.Message,"Only the first") {
					// Only the first 1000 search results are available
					PrintInfos("debug", "search limit reached")
					break
				} else if strings.HasPrefix(r.Message,"Bad credentials") {
					// Bad credentials
					config.tokens = resliceTokens( config.tokens, token_index )
					n_token--
				} else if strings.HasPrefix(r.Message,"You have triggered an abuse detection mechanism") {
					// You have triggered an abuse detection mechanism. Please wait a few minutes before you try again.
					PrintInfos("debug", "token limit reached, token disabled")
					config.tokens[token_index].disabled_ts = time.Now().Unix() + 70
				}
			}

			if page == 1 {
				t_search[search_index].TotalCount = r.TotalCount
				max_page = int( math.Ceil( float64(t_search[search_index].TotalCount)/100.00 ) )
				if max_page > 10 {
					max_page = 10
				}

				if r.TotalCount > 1000 {
					if( config.quick_mode ) {
						// if search_index == 0 {
						// 	t_search = append( t_search, Search{keyword:config.search, sort:"indexed", order:"asc"} )
						// 	t_search = append( t_search, Search{keyword:config.search, sort:"", order:"desc"} )
						// 	PrintInfos( "debug", fmt.Sprintf("current search returned %d results, extra searches added",t_search[search_index].TotalCount) )
						// }
					} else {
						if current_search.language == "" && len(t_languages) > 0 {
							addSearchLanguage( current_search )
							PrintInfos( "debug", fmt.Sprintf("current search returned %d results, language filter added for later search",t_search[search_index].TotalCount) )
						} else if len(t_noise) > 0 {
							addSearchNoise( current_search )
							PrintInfos( "debug", fmt.Sprintf("current search returned %d results, noise added for later search",t_search[search_index].TotalCount) )
						}
					}
					n_search = len(t_search)
				} else {
					PrintInfos( "debug", fmt.Sprintf("current search returned %d results", t_search[search_index].TotalCount) )
				}
			}

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
		}

		search_index++
	}

	PrintInfos( "", fmt.Sprintf("%d searches performed",n_search) )
	PrintInfos( "", fmt.Sprintf("%d subdomains found",len(t_subdomains)) )
}


func addSearchLanguage( current_search Search ) {

	for _,language := range t_languages {
		var new_search Search
		new_search.keyword = current_search.keyword
		new_search.sort = current_search.sort
		new_search.order = current_search.order
		new_search.language = language
		new_search.signature = generateSignature( new_search )
		t_search = append( t_search, new_search )
	}
}


func addSearchNoise( current_search Search ) {

	for _,noise := range t_noise {
		if !inArray(noise,current_search.noise) {
			var new_search Search
			new_search.keyword = current_search.keyword
			new_search.sort = current_search.sort
			new_search.order = current_search.order
			new_search.language = current_search.language
			new_search.noise = append( current_search.noise, noise )
			new_search.signature = generateSignature( new_search )
			if !searchExists(new_search.signature) {
				// PrintInfos( "debug", fmt.Sprintf("search added because signature not found %s",new_search.signature) )
				t_search = append( t_search, new_search )
			} else {
				// PrintInfos( "debug", fmt.Sprintf("search NOT added because signature WAS found %s",new_search.signature) )
			}
		}
	}
}


func searchExists( signature string ) bool {
	for _,search := range t_search {
		if signature == search.signature {
			return true
		}
	}
	return false
}


func generateSignature( s Search ) string {

	var tab = []string{ s.keyword, s.language }
	sort.Strings(s.noise)
	tab = append( tab, s.noise... )

	return GetMD5Hash( strings.Join(tab,"||") )

}


func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}


func inArray(str string, array []string) bool {
	for _,i := range array {
		if i == str {
			return true
		}
	}
	return false
}


func performRegexp(code string, rgxp *regexp.Regexp ) [][]byte {
	return rgxp.FindAll([]byte(code), -1)
}


func getRawUrl( html_url string ) string {
    var raw_url = html_url
    raw_url = strings.Replace( raw_url, "https://github.com/", "https://raw.githubusercontent.com/", -1 )
    raw_url = strings.Replace( raw_url, "/blob/", "/", -1 )
	return raw_url
}


func resliceTokens(s []Token, index int) []Token {
    return append(s[:index], s[index+1:]...)
}


func displayConfig() {
	PrintInfos( "", fmt.Sprintf("Domain:%s, Output:%s",config.domain,config.output) )
	PrintInfos( "", fmt.Sprintf("Tokens:%d, Delay:%.0fms",len(config.tokens),float32(config.delay)) )
	PrintInfos( "", fmt.Sprintf("Token rehab:%t, Quick mode:%t",!config.stop_notoken,config.quick_mode) )
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
	fmt.Print(`
	   ▗▐  ▌     ▌          ▌    ▌          ▗
	▞▀▌▄▜▀ ▛▀▖▌ ▌▛▀▖  ▞▀▘▌ ▌▛▀▖▞▀▌▞▀▖▛▚▀▖▝▀▖▄ ▛▀▖▞▀▘
	▚▄▌▐▐ ▖▌ ▌▌ ▌▌ ▌  ▝▀▖▌ ▌▌ ▌▌ ▌▌ ▌▌▐ ▌▞▀▌▐ ▌ ▌▝▀▖
	▗▄▘▀▘▀ ▘ ▘▝▀▘▀▀   ▀▀ ▝▀▘▀▀ ▝▀▘▝▀ ▘▝ ▘▝▀▘▀▘▘ ▘▀▀
	`)
	fmt.Print("       by @gwendallecoguic                          \n\n")
}
