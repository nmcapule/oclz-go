package opencart

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
)

var pagesRe = regexp.MustCompile(`(?P<offset>\d+) to (?P<offset_limit>\d+) of (?P<total>\d+) \((?P<pages>\d+) Pages\)`)

func catalogProductParser(input string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
	if err != nil {
		log.Fatalf("parsing doc: %v", err)
	}
	var rows []map[string]interface{}
	doc.Find("#form-product > div > table > tbody > tr").Each(func(_ int, s *goquery.Selection) {
		rows = append(rows, map[string]interface{}{
			"model":        strings.TrimSpace(s.Find("td:nth-child(4)").Text()),
			"quantity":     strings.TrimSpace(s.Find("td:nth-child(6) > span").Text()),
			"product_name": strings.TrimSpace(s.Find("td:nth-child(3)").Text()),
			"price":        strings.TrimSpace(s.Find("td:nth-child(5)").Text()),
			"status":       strings.TrimSpace(s.Find("td:nth-child(7)").Text()),
		})
	})
	tokens := pagesRe.FindStringSubmatch(doc.Find("#form-product + div > div + div").Text())
	offset, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("offset")])
	offsetLimit, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("offset_limit")])
	total, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("total")])
	pages, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("pages")])
	// TODO(nmcapule): Handle other errors.
	b, err := json.Marshal(map[string]interface{}{
		"code":    0,
		"message": "Success",
		"data": map[string]interface{}{
			"rows":   rows,
			"offset": offset - 1,
			"limit":  offsetLimit - offset + 1,
			"total":  total,
			"pages":  pages,
		},
	})
	if err != nil {
		log.Fatalf("encoding rows: %v", err)
	}
	return string(b)
}
