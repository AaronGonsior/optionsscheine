package optionsscheine

import (
    "os"
    "bufio"
    "io/ioutil"
    "encoding/json"
    "net/http"
    "net/url"
    "fmt"
    "strconv"
    "github.com/PuerkitoBio/goquery"
    "strings"
    "time"
    "bytes"
    "errors"
)

type Stock struct {
    Isin   string
    Ticker string
    Name   string
}

type sa_stock struct {
    Ticker string `json:"s"`
    Name   string `json:"n"`
    Info   string `json:"i"`
    Date   uint   `json:"m"`
}

var sa_stocks []sa_stock

func (stock *Stock) Complete() {

    if len(sa_stocks) == 0 {
        client := &http.Client{}
        r, _ := http.NewRequest(http.MethodGet, "https://stockanalysis.com/stocks/", nil)
        r.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
        r.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.51 Safari/537.36")

        resp, _ := client.Do(r)

        content, _ := ioutil.ReadAll(resp.Body)

        html := string(content)

        html = strings.Split(html, "type=\"application/json\">")[1]
        json_string := strings.Split(html, "</script>")[0]
        //fmt.Println(json_string)

        json_aa := getalt([]byte(json_string), "props")
        json_aa = getalt([]byte(json_aa), "pageProps")
        json_aa = getalt([]byte(json_aa), "stocks")

        json.Unmarshal([]byte(json_aa), &sa_stocks)
    }

    if stock.Isin != "" && stock.Ticker == ""{

        client := &http.Client{}
        r, _ := http.NewRequest(http.MethodGet, "https://stockmarketmba.com/lookupisinonopenfigi.php", nil)
        
        resp, _ := client.Do(r)

        cookie := resp.Cookies()

        content, _ := ioutil.ReadAll(resp.Body)
        defer resp.Body.Close()

        html := string(content)
        html = strings.Split(html, "name='version' value=\"")[1]
        version := strings.Split(html, "\">")[0]

        data := url.Values{}
        data.Add("action", "Go")
        data.Add("version", version)
        data.Add("search", stock.Isin)

        client = &http.Client{}
        r, _ = http.NewRequest(http.MethodPost, "https://stockmarketmba.com/lookupisinonopenfigi.php", strings.NewReader(data.Encode()))  // URL-encoded payload
        r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
        r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
        r.Header.Add("Host", "stockmarketmba.com")
        r.Header.Add("Origin", "https://stockmarketmba.com")
        r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.120 Safari/537.36")
        r.Header.Add("Referer", "https://stockmarketmba.com/lookupisinonopenfigi.php")
        
        for i := range cookie {
            r.AddCookie(cookie[i])
        }

        resp, _ = client.Do(r)

        content, _ = ioutil.ReadAll(resp.Body)
        defer resp.Body.Close()
        
        html = string(content)


        htmlsplit := strings.Split(html, "<tbody><tr><td>")
        if len(htmlsplit) > 1 {
            html = htmlsplit[1]
            ticker := strings.Split(html, "</td><td>")[0]
            stock.Ticker = ticker
        }

    }

    if stock.Ticker == "" && stock.Name != "" {
        stock.Ticker = findtickerbyname(sa_stocks, stock.Name)
    }

    if stock.Name == "" && stock.Ticker != "" {
        stock.Name = findnamebyticker(sa_stocks, stock.Ticker)
    }


}

func getalt(json_byte []byte, name string) string{
    var v interface{}
    json.Unmarshal(json_byte, &v)
    data := v.(map[string]interface{})
    json_alt, _ := json.Marshal(data[name])

    return string(json_alt)
}


func findnamebyticker(stocks []sa_stock, itemticker string) string {
    for _, item := range stocks {
        if strings.ToLower(item.Ticker) == strings.ToLower(itemticker) {
            return item.Name
        }
    }
    return ""
}

func findtickerbyname(stocks []sa_stock, itemname string) string {
    for _, item := range stocks {
        if strings.ToLower(item.Name) == strings.ToLower(itemname) {
            return item.Ticker
        }
    }
    return ""
}

const hsbc = "hsbc"
const bnpparibas = "bnpparibas"

type Date struct {
    Day int
    Month int
    Year int
}

type Call struct {
    Name string
    Wkn string
    Isin string
    Strike float64
    Ask float64
    Bid float64
    Factor float64
    Date Date
    CallType string
    Bank string
}

func (t Call) GetFactor() Call {
    var result Call = t

    if t.Bank == hsbc &&  t.Factor == 0.0 {
        result = get_by_isin_hsbc(t.Isin)
    } 

    return result

}

type Option_search struct {
    Stockk           Stock
    Strike_range   []int
    Exp_date_range []Date
    CallType       string
    Bank           string
}

var cookie_bnpparibas []*http.Cookie
var cookie_hsbc []*http.Cookie
var sessionid_hsbc string

var name_lookup_hsbc map[string]string = make(map[string]string)
var name_lookup_bnp  map[string]string = make(map[string]string)


func (t Option_search) Find() ([]Call, error){

    var results []Call
    
    if t.Stockk.Name == "" && t.Stockk.Isin == "" {
        return results, errors.New("name and isin parameters not be nill same time")
    } else if len(t.Strike_range) > 2 {
        return results, errors.New("Strike_range parameter can take at most two value")
    } else if len(t.Exp_date_range) > 2 {
        return results, errors.New("Exp_date_range parameter can take at most two value")
    } else if t.CallType != "call" &&  t.CallType != "put" &&  t.CallType != "" {
        return results, errors.New("CallType parameter call, put and \"\" values")
    } else if t.Bank != "hsbc" &&  t.Bank != bnpparibas &&  t.Bank != "" {
        return results, errors.New("Bank parameter hsbc, bnpparibas and \"\" values")
    } else if t.Bank == bnpparibas {
        results = findforbnp(t)
    } else if t.Bank == "hsbc" {
        results = findforhsbc(t)
    } else if t.Bank == "" {
        results = append(findforhsbc(t), findforbnp(t)...)
    }
    return results, nil
}

func Get_by_isin(isin string) Call {
    banks := []string{"hsbc","bnpparibas"}
    for _,bank := range banks {
        tmp := Get_by_isin_by_bank(isin,bank)
        if tmp.Name != " " && tmp.Isin != "" {
            return tmp
        }
    }
    return Call{}
}

func Get_by_isin_by_bank(isin string, bank string) Call {
    var result Call

    if bank == hsbc {
        result =  get_by_isin_hsbc(isin)
    } else if bank == bnpparibas {
        result =  get_by_isin_bnp(isin)
    }

    return result
}

func findforbnp(t Option_search) []Call {
    var offset, limit, total int
    id, class := getidbyName(strings.ToLower(t.Stockk.Name))
    var results []Call

    jsonStr := createJson(id, class, t.Strike_range, t.Exp_date_range, t.CallType, offset)
    results, offset, limit, total = getbyJson(jsonStr, results)
    for offset < total - limit {
        offset = offset + limit
        jsonStr = createJson(id, class, t.Strike_range, t.Exp_date_range, t.CallType, offset)
        results, offset, limit, total = getbyJson(jsonStr, results) 
    }

    return results
}
func findforhsbc(t Option_search) []Call {

    if len(name_lookup_hsbc) == 0 {
        f, _ := os.Open("hsbcnamelookup")
        defer f.Close()

        scanner := bufio.NewScanner(f)

        for scanner.Scan() {
            line := scanner.Text()
            resss := strings.Split(line, ",")
            name_lookup_hsbc[resss[0]] = resss[1]
        } 
    }

    var url string = "https://www.hsbc-zertifikate.de/home/produkte/hebelprodukte/optionsscheine#!/filter:"

    var id = name_lookup_hsbc[strings.ToLower(t.Stockk.Name)]

    url = url+"underlyings=" + id+ "/"

    if t.CallType == "call" {
        url = url + "is-call/isnot-put/"
    } else if t.CallType == "put" {
        url = url + "isnot-call/is-put/"
    } else {
        url = url + "is-call/is-put/"
    }
        
    if len(t.Strike_range)==2 {
        url = url + "strike_from=" + strconv.Itoa(t.Strike_range[0]) + ";strike_to=" + strconv.Itoa(t.Strike_range[1]) + "/"
    } else if len(t.Strike_range)==1 {
        url = url + "strike_from=" + strconv.Itoa(t.Strike_range[0]) + ";strike_to=" + strconv.Itoa(t.Strike_range[0]) + "/"
    }

    if len(t.Exp_date_range)==2 {
        url = url + "expiry_from=" + date_to_string(t.Exp_date_range[0]) + ";expiry_to=" + date_to_string(t.Exp_date_range[1]) + "/"
    } else if len(t.Exp_date_range)==1 {
        url = url + "expiry_from=" + date_to_string(t.Exp_date_range[0]) + ";expiry_to=" + date_to_string(t.Exp_date_range[0]) + "/"
    }

    url = url + "pn=0;ps=40000;sort=expiry;ascending=true"
    //fmt.Println(url)
    return searchforhsbc(url)
}

func searchforhsbc(search_url string) []Call {
    var results []Call
    var result Call
    var res map[string]interface{}
    var name, wkn, isin, callType, delta, omega string
    var strike, bid, ask float64
    var day, mount, year int

    if len(cookie_hsbc) == 0 {
        cookie_hsbc, sessionid_hsbc = getCookieforHSBC() 
    }

    tmilliseconds := getTime()
    data := url.Values{}
    data.Add("v-browserDetails", "1")
    data.Add("theme", "hsbc")
    data.Add("v-appId", "myApp")
    data.Add("v-sh", "1440")
    data.Add("v-sw", "2560")
    data.Add("v-cw", "869")
    data.Add("v-ch", "1006")
    data.Add("v-curdate", tmilliseconds)
    data.Add("v-tzo", "-180")
    data.Add("v-dstd", "0")
    data.Add("v-rtzo", "-180")
    data.Add("v-dston", "false")
    data.Add("v-vw", "50")
    data.Add("v-vh", "50")
    data.Add("v-loc", search_url)
    data.Add("v-wn", "myApp-0.9458672729469536")

    client := &http.Client{}
    r, _ := http.NewRequest(http.MethodPost, "https://www.hsbc-zertifikate.de/web-htde-tip-zertifikate-main/?components=c2VhcmNoaGludF9tb2JpbGU6U2VhcmNoSGludE1vYmlsZUNvbXBvbmVudCgndWxTZWFyY2hTbWFsbC9zZWFyY2hJbnB1dE1vYmlsZScpO3NlYXJjaGhpbnQ6U2VhcmNoSGludENvbXBvbmVudCgndWxTZWFyY2hGdWxsL3NlYXJjaC1oZWFkZXInKTtwYWdlLXRpdGxlOlByb2R1Y3RQYWdlRGVzY3JpcHRpb25MYWJlbCgnU3RhbmRhcmQtT3B0aW9uc3NjaGVpbmUnKTthbXBlbDpSdFB1bGxDb21wb25lbnQoJ2FuaW1Dc3MsYy1oaWdobGlnaHQtdXAsYy1oaWdobGlnaHQtZG93bixjLWhpZ2hsaWdodC1jaGFuZ2VkJyk7ZmlsdGVyOlJlc3BvbnNpdmVGaWx0ZXJlZFRhYmxlKCdQYWdlUGxhaW5Pcycp&pagepath="+url.QueryEscape(search_url)+"&magnoliaSessionId="+sessionid_hsbc+"&v-"+tmilliseconds, strings.NewReader(data.Encode()))  // URL-encoded payload
    r.Header.Set("Accept", "*/*")
    r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    r.Header.Add("Origin", "https://www.hsbc-zertifikate.de")
    for i := range cookie_hsbc {
        r.AddCookie(cookie_hsbc[i])
    }
    r.Header.Add("Host", "www.hsbc-zertifikate.de")
    r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.120 Safari/537.36")
    r.Header.Add("Referer", "https://www.hsbc-zertifikate.de/home/produkte/hebelprodukte/optionsscheine")

    resp2, _ := client.Do(r)
    body, _ := ioutil.ReadAll(resp2.Body)
    defer resp2.Body.Close()

    //fmt.Println(string(body))

    
    if resp2.StatusCode == 200 {
        _ = json.Unmarshal([]byte(body), &res)
        _ = json.Unmarshal([]byte(res["uidl"].(string)), &res)

        for _, element := range res["state"].(map[string]interface{}) {        

            if element.(map[string]interface{})["contentMode"] == "HTML" && element.(map[string]interface{})["text"] != nil && element.(map[string]interface{})["text"].(string)[1:6] == "table" {

                table := element.(map[string]interface{})["text"].(string)
                doc, err := goquery.NewDocumentFromReader(strings.NewReader(table))
                if err != nil {
                  fmt.Println(err)
                }
                doc.Find("tr").Each(func(i1 int, s1 *goquery.Selection) {
                    // For each item found, get the title
                    s1.Find("td").Each(func(i2 int, s2 *goquery.Selection) {
                        // For each item found, get the title
                        if i2 == 1 {
                            name = s2.Text()
                        } else if i2 == 2 {
                            wkn = s2.Text()
                        } else if i2 == 4 {
                            callType = strings.ToLower(s2.Text())
                        } else if i2 == 5 {
                            t, _ := time.Parse("02.01.06", s2.Text())
                            day, mount, year = t.Day(), int(t.Month()), t.Year()
                        } else if i2 == 6 {
                            strike = fixPrice(s2.Text())
                            //strike = s2.Text()
                        } else if i2 == 7 {
                            bid = fixPrice(s2.Text())
                        } else if i2 == 8 {
                            ask = fixPrice(s2.Text())
                        } else if i2 == 9 {
                            delta = s2.Text()
                        } else if i2 == 10 {
                            omega = s2.Text()
                        } else if i2 == 12 {
                            temp, _ := s2.Attr("data-expandable-id")
                            isin = temp
                        }
                    })
                    result = Call{name, wkn, isin, strike, ask, bid, 0.0, Date{day, mount, year}, callType, hsbc}
                    if wkn != "" { results = append(results, result) }
                    
                })
            }
        }
        
    } else {
        results = append(results, result)
    }

    return results
}

func get_by_isin_bnp(isin string) Call {
    var result Call
    var res map[string]interface{}
    var ratio float64
    var day, mount, year int

    if len(cookie_bnpparibas) == 0 {
        cookie_bnpparibas = getCookieforBNP() 
    }

    client := &http.Client{}
    r, _ := http.NewRequest(http.MethodGet, "https://derivate.bnpparibas.com/apiv2/api/v1/product/header/"+isin, nil)
    
    r.Header.Set("Accept", "*/*")
    r.Header.Add("clientid", "0")
    r.Header.Add("Connection", "keep-alive")
    r.Header.Add("Content-Type", "application/json")
    for i := range cookie_bnpparibas {
        r.AddCookie(cookie_bnpparibas[i])
    }
    r.Header.Add("Host", "derivate.bnpparibas.com")
    r.Header.Add("Referer", "https://derivate.bnpparibas.com/product-details/"+isin+"/")
    r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.120 Safari/537.36")
    

    resp2, _ := client.Do(r)
    body, _ := ioutil.ReadAll(resp2.Body)

    defer resp2.Body.Close()

    if resp2.StatusCode == 200 {
        _ = json.Unmarshal([]byte(body), &res)

        t, _ := time.Parse("2006-01-02T15:04:05", res["result"].(map[string]interface{})["keyFigures"].(map[string]interface{})["maturityDate"].(string))
        day, mount, year = t.Day(), int(t.Month()), t.Year()

        _, ok := res["result"].(map[string]interface{})["first"].(map[string]interface{})["ratio"]
        if ok {
                ratio = res["result"].(map[string]interface{})["first"].(map[string]interface{})["ratio"].(float64)
        } else {
            ratio = 0.0
        }

        result = Call{res["result"].(map[string]interface{})["firstUnderlyingName"].(string), res["result"].(map[string]interface{})["wkn"].(string), res["result"].(map[string]interface{})["isin"].(string), res["result"].(map[string]interface{})["first"].(map[string]interface{})["strikeAbsolute"].(float64), res["result"].(map[string]interface{})["ask"].(float64), res["result"].(map[string]interface{})["bid"].(float64), ratio, Date{day, mount, year}, res["result"].(map[string]interface{})["derivativeTypeName"].(string), bnpparibas}

    }

    return result
}

func get_by_isin_hsbc(isin string) Call {

    var result Call
    var res map[string]interface{}

    if len(cookie_hsbc) == 0 {
        cookie_hsbc, sessionid_hsbc = getCookieforHSBC() 
    }

    tmilliseconds := getTime()

    data := url.Values{}
    data.Add("v-browserDetails", "1")
    data.Add("theme", "hsbc")
    data.Add("v-appId", "myApp")
    data.Add("v-sh", "1440")
    data.Add("v-sw", "2560")
    data.Add("v-cw", "869")
    data.Add("v-ch", "1006")
    data.Add("v-curdate", tmilliseconds)
    data.Add("v-tzo", "-180")
    data.Add("v-dstd", "0")
    data.Add("v-rtzo", "-180")
    data.Add("v-dston", "false")
    data.Add("v-vw", "50")
    data.Add("v-vh", "50")
    data.Add("v-loc", "https://www.hsbc-zertifikate.de/home/details#!/isin:"+isin)
    data.Add("v-wn", "myApp-0.3035392384152932")

    client := &http.Client{}
    r, _ := http.NewRequest(http.MethodPost, "https://www.hsbc-zertifikate.de/web-htde-tip-zertifikate-main/?components=YW1wZWw6UnRQdWxsQ29tcG9uZW50KCdhbmltQ3NzLGMtaGlnaGxpZ2h0LXVwLGMtaGlnaGxpZ2h0LWRvd24sYy1oaWdobGlnaHQtY2hhbmdlZCcpO3NlYXJjaGhpbnRfbW9iaWxlOlNlYXJjaEhpbnRNb2JpbGVDb21wb25lbnQoJ3VsU2VhcmNoU21hbGwvc2VhcmNoSW5wdXRNb2JpbGUnKTtzZWFyY2hoaW50OlNlYXJjaEhpbnRDb21wb25lbnQoJ3VsU2VhcmNoRnVsbC9zZWFyY2gtaGVhZGVyJyk7aXNpbjpSZXNwb25zaXZlU25hcHNob3RDb21wb25lbnQoJ2ZhbHNlJyk%3D&pagepath=https%3A%2F%2Fwww.hsbc-zertifikate.de%2Fhome%2Fdetails%23!%2Fisin%3A"+isin+"&magnoliaSessionId="+sessionid_hsbc+"&v-"+tmilliseconds, strings.NewReader(data.Encode()))  // URL-encoded payload
    
    r.Header.Set("Accept", "*/*")
    r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    r.Header.Add("Origin", "https://www.hsbc-zertifikate.de")
    for i := range cookie_hsbc {
        r.AddCookie(cookie_hsbc[i])
    }
    r.Header.Add("Host", "www.hsbc-zertifikate.de")
    r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.120 Safari/537.36")
    r.Header.Add("Referer", "https://www.hsbc-zertifikate.de/home/details")

    resp2, _ := client.Do(r)
    body, _ := ioutil.ReadAll(resp2.Body)
    defer resp2.Body.Close()

    var name, wkn, callType string
    var strike, factor, ask, bid float64
    var day, mount, year int

    if resp2.StatusCode == 200 {
        _ = json.Unmarshal([]byte(body), &res)
        _ = json.Unmarshal([]byte(res["uidl"].(string)), &res)

         for _, element := range res["state"].(map[string]interface{}) {        

            if element.(map[string]interface{})["contentMode"] == "HTML" && element.(map[string]interface{})["text"] != nil {

                divs := element.(map[string]interface{})["text"].(string)
                docs, _ := goquery.NewDocumentFromReader(strings.NewReader(divs))


                docs.Find("div").Each(func(i1 int, s1 *goquery.Selection) {

                    s1.Find("td").Each(func(i2 int, s2 *goquery.Selection) {
                        // For each item found, get the title
                       //aa := s2.Text()
                        if s2.Text() == "Basiswert" {
                           name = s2.Next().Text() 
                        } else if s2.Text() == "WKN" {
                           wkn = s2.Next().Text() 
                        } else if s2.Text() == "Basispreis" {
                           strike = fixPrice(s2.Next().Text()) 
                        } else if s2.Text() == "Bezugsverhältnis" {
                           factor = fixPrice(s2.Next().Text()) 
                        } else if s2.Text() == "Optionsscheintyp" {
                           callType = strings.ToLower(s2.Next().Text() )
                        } else if strings.HasPrefix(s2.Text(), "Geldkurs") {
                           bid = fixPrice(s2.Next().Text()) 
                        } else if strings.HasPrefix(s2.Text(), "Briefkurs") {
                           ask = fixPrice(s2.Next().Text()) 
                        } else if strings.HasPrefix(s2.Text(), "Letzter Tag der Ausübungsfrist") {
                           t, _ := time.Parse("02.01.06", s2.Next().Text()[:8])
                           day, mount, year = t.Day(), int(t.Month()), t.Year()
                        }

                    })
                    
                })

                result = Call{name, wkn, isin, strike, ask, bid, factor, Date{day, mount, year}, callType, hsbc}
            } 
        }


        
    }

    return result
}
func date_to_string(date Date) string {
    month  := strconv.Itoa(date.Month)
    day := strconv.Itoa(date.Day)
    if len(month) == 1 { month = "0"+month }
    if len(day) == 1 { day = "0"+day }
    return strconv.Itoa(date.Year)+"-"+month+"-"+day
}

func merge(start []byte, nums ...[]byte) []byte {
    merged := start
    for _, num := range nums {
        merged = append(merged, num...)
    }
    return merged
}

func createJson(id int, class int, strike_range []int, exp_date_range []Date, callType string, offset int) []byte {
        var optiontype, optionstrike, optiondate, jsonend []byte
        var jsonpre = []byte(`{"clientId":0,"sortPreference":null,"derivativeTypeIds":[11,12],"productGroupIds":[3],"productSetIds":[],"filterOptions":[`)
        
        var option01 = merge([]byte(`{"fieldKey":"firstAssetClassId","filterType":"DropDown","switchKey":null,"hideColumnNameWhenFilterHasValue":null,"displayOrder":1,"url":null,"selectedValues":["`), []byte(strconv.Itoa(class)), []byte(`"]},`))
        var optionId = merge([]byte(`{"fieldKey":"firstUnderlyingId","filterType":"DropDown","switchKey":null,"hideColumnNameWhenFilterHasValue":"firstUnderlyingName","displayOrder":2,"url":"quicksearch/underlyings","selectedValues":["`), []byte(strconv.Itoa(id)), []byte(`"]},`))

        if callType == "call" {
            optiontype = []byte(`{"fieldKey":"list-isCall","filterType":"RadioButtonSwitch","switchKey":"Call_Put","hideColumnNameWhenFilterHasValue":null,"displayOrder":3,"url":null,"checkBoxValue":true},{"fieldKey":"list-isPut","filterType":"RadioButtonSwitch","switchKey":"Call_Put","hideColumnNameWhenFilterHasValue":null,"displayOrder":4,"url":null,"checkBoxValue":false},`)
        } else if callType == "put" {
            optiontype = []byte(`{"fieldKey":"list-isCall","filterType":"RadioButtonSwitch","switchKey":"Call_Put","hideColumnNameWhenFilterHasValue":null,"displayOrder":3,"url":null,"checkBoxValue":false},{"fieldKey":"list-isPut","filterType":"RadioButtonSwitch","switchKey":"Call_Put","hideColumnNameWhenFilterHasValue":null,"displayOrder":4,"url":null,"checkBoxValue":true},`)
        } else {
            optiontype = []byte(`{"fieldKey":"list-isCall","filterType":"RadioButtonSwitch","switchKey":"Call_Put","hideColumnNameWhenFilterHasValue":null,"displayOrder":3,"url":null},{"fieldKey":"list-isPut","filterType":"RadioButtonSwitch","switchKey":"Call_Put","hideColumnNameWhenFilterHasValue":null,"displayOrder":4,"url":null},`)
        }

        var option05 = []byte(`{"fieldKey":"list-issuedToday","filterType":"Switch","switchKey":null,"hideColumnNameWhenFilterHasValue":null,"displayOrder":5,"url":null},`)
        var option06 = []byte(`{"fieldKey":"derivativeTypeId","filterType":"CheckBoxGroup","switchKey":null,"hideColumnNameWhenFilterHasValue":null,"displayOrder":6,"url":null,"selectedValues":[11,12]},`)
        

        if len(strike_range)==2 {
            optionstrike = merge([]byte(`{"fieldKey":"first.strikeAbsolute","filterType":"Range","displayOrder":7,"selectedRange":[`), []byte(strconv.Itoa(strike_range[0])), []byte(`,`), []byte(strconv.Itoa(strike_range[1])), []byte(`]},`))
        } else if len(strike_range)==1 {
            optionstrike = merge([]byte(`{"fieldKey":"first.strikeAbsolute","filterType":"Range","displayOrder":7,"selectedRange":[`), []byte(strconv.Itoa(strike_range[0])), []byte(`,`), []byte(strconv.Itoa(strike_range[0])), []byte(`]},`))
        } else {
            optionstrike = []byte(`{"fieldKey":"first.strikeAbsolute","filterType":"Range","displayOrder":7},`)
        }

        if len(exp_date_range)==2 {
            optiondate = merge([]byte(`{"fieldKey":"keyFigures.maturityDate","filterType":"DateRangePicker","switchKey":null,"hideColumnNameWhenFilterHasValue":null,"displayOrder":8,"url":null,"selectedDateRange":["`), []byte(time.Date(exp_date_range[0].Year, time.Month(exp_date_range[0].Month), exp_date_range[0].Day, 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05")), []byte(`","`), []byte(time.Date(exp_date_range[1].Year, time.Month(exp_date_range[1].Month), exp_date_range[1].Day, 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05")), []byte(`"]},`))
        } else if len(exp_date_range)==1 {
            optiondate = merge([]byte(`{"fieldKey":"keyFigures.maturityDate","filterType":"DateRangePicker","switchKey":null,"hideColumnNameWhenFilterHasValue":null,"displayOrder":8,"url":null,"selectedDateRange":["`), []byte(time.Date(exp_date_range[0].Year, time.Month(exp_date_range[0].Month), exp_date_range[0].Day, 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05")), []byte(`","`), []byte(time.Date(exp_date_range[0].Year, time.Month(exp_date_range[0].Month), exp_date_range[0].Day, 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05")), []byte(`"]},`))
        } else {
            optiondate = []byte(`{"fieldKey":"keyFigures.maturityDate","filterType":"DateRangePicker","switchKey":null,"hideColumnNameWhenFilterHasValue":null,"displayOrder":8,"url":null},`)
        }

        var option0910 = []byte(`{"fieldKey":"delta","filterType":"Range","displayOrder":9},{"fieldKey":"omega","filterType":"Range","displayOrder":10}`)
        
        if offset == 0 {
            jsonend = []byte(`],"firstUnderlyingIsin":null,"isBNL":null,"isDB":null,"responsetype":null,"productFlagFilter":-1,"isDirectionFilterCanBeDisabled":true,"queryString":null}`)
        } else {
            jsonend = merge([]byte(`],"firstUnderlyingIsin":null,"isBNL":null,"isDB":null,"responsetype":null,"productFlagFilter":-1,"isDirectionFilterCanBeDisabled":true,"queryString":null,"offset":`), []byte(strconv.Itoa(offset)), []byte(`}`))
        }
        

        jsonStr := merge(jsonpre, option01, optionId, optiontype, option05, option06, optionstrike, optiondate, option0910, jsonend)

        return jsonStr
}


func getbyJson(jsonStr []byte, results []Call) ([]Call, int, int, int) {

    //cookie_bnpparibas = getCookieforBNP()


    if len(cookie_bnpparibas) == 0 {
        cookie_bnpparibas = getCookieforBNP() 
    }

    var result Call
    var ratio float64
    var day, mount, year int
    var res map[string]interface{}
    var res2 map[string]interface{}

    client := &http.Client{}
    r, _ := http.NewRequest(http.MethodPost, "https://derivate.bnpparibas.com/apiv2/api/v1/productlist/", bytes.NewBuffer(jsonStr))  // URL-encoded payload
    r.Header.Set("Accept", "*/*")
    r.Header.Add("Content-Type", "application/json")
    for i := range cookie_bnpparibas {
        r.AddCookie(cookie_bnpparibas[i])
    }
    r.Header.Set("Accept", "*/*")
    r.Header.Add("Host", "derivate.bnpparibas.com")
    r.Header.Add("Origin", "https://derivate.bnpparibas.com")
    r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.120 Safari/537.36")
    r.Header.Add("Referer", "https://derivate.bnpparibas.com/optionsscheine/")
    r.Header.Add("clientid", "0")

    resp2, _ := client.Do(r)
    body, _ := ioutil.ReadAll(resp2.Body)

    defer resp2.Body.Close()

    var offset, limit, total int
    if resp2.StatusCode == 200 {
        _ = json.Unmarshal([]byte(body), &res)

        offset = int(res["offset"].(float64))
        limit  = int(res["limit"].(float64))
        total  = int(res["total"].(float64))
        for _, str := range res["result"].([]interface{}) {
            res2 = str.(map[string]interface{})
                
            t, _ := time.Parse("2006-01-02T15:04:05", res2["keyFigures"].(map[string]interface{})["maturityDate"].(string))
            day, mount, year = t.Day(), int(t.Month()), t.Year()
                

            _, ok := res2["first"].(map[string]interface{})["ratio"]

            if ok {
                        ratio = res2["first"].(map[string]interface{})["ratio"].(float64)
            } else {
                        ratio = 0.0
            }

            result = Call{res2["name"].(string), res2["wkn"].(string), res2["isin"].(string), res2["first"].(map[string]interface{})["strikeAbsolute"].(float64), res2["ask"].(float64), res2["bid"].(float64), ratio, Date{day, mount, year}, res2["derivativeTypeName"].(string), bnpparibas}
            results = append(results, result)
        }
    }
    return results, offset, limit, total
 }


func fixPrice(input string) float64 {
    filter := func(r rune) rune {
        if strings.IndexRune("USD EUR .", r) < 0 {
            return r
        }
        return -1
    }
    str := strings.Map(filter, input)
    str = strings.ReplaceAll(str, ",", ".")
    f, _ := strconv.ParseFloat(str, 64)
    return f
 }


func getCookieforBNP() []*http.Cookie {
    resp, _ := http.Get("https://derivate.bnpparibas.com/optionsscheine/")
    return resp.Cookies()
}
func getCookieforHSBC() ([]*http.Cookie, string) {
    resp, _ := http.Get("https://www.hsbc-zertifikate.de/")
    
    for _, cookie := range resp.Cookies() {
        if cookie.Name == "JSESSIONID" {
            return resp.Cookies(), cookie.Value
        }
    }

    return resp.Cookies(), ""
}

func getTime() string {
    tmilliseconds:= int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
    return strconv.FormatInt(tmilliseconds, 10)
}

func getidbyName(name string) (int, int) {
    var resultid,  resultclass = 0, 0

    var res map[string]interface{}

    if len(cookie_bnpparibas) == 0 {
        cookie_bnpparibas = getCookieforBNP() 
    }

    client := &http.Client{}
    r, _ := http.NewRequest(http.MethodGet, "https://derivate.bnpparibas.com/apiv2/api/v1/quicksearch/underlyings", nil)
    
    r.Header.Set("Accept", "*/*")
    r.Header.Add("clientid", "0")
    r.Header.Add("Connection", "keep-alive")
    r.Header.Add("Content-Type", "application/json")
    for i := range cookie_bnpparibas {
        r.AddCookie(cookie_bnpparibas[i])
    }
    r.Header.Add("Host", "derivate.bnpparibas.com")
    r.Header.Add("Referer", "https://derivate.bnpparibas.com/optionsscheine/")
    r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.120 Safari/537.36")
    
    query := r.URL.Query()
    query.Add("term", name)
    r.URL.RawQuery = query.Encode()

    resp2, _ := client.Do(r)
    body, _ := ioutil.ReadAll(resp2.Body)

    defer resp2.Body.Close()

    if resp2.StatusCode == 200 {
        _ = json.Unmarshal([]byte(body), &res)
        resultid = int(res["results"].([]interface{})[0].(map[string]interface{})["id"].(float64))
        resultclass = int(res["results"].([]interface{})[0].(map[string]interface{})["assetClass"].(float64))
    }

    return resultid, resultclass
}

