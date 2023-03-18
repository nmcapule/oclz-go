package opencart

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/nmcapule/oclz-go/utils"
	"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

var pagesRe = regexp.MustCompile(`(?P<offset>\d+) to (?P<offset_limit>\d+) of (?P<total>\d+) \((?P<pages>\d+) Pages\)`)

var MessageNoResults = "No results!"

func scrapeCatalogProduct(input string) (*gjson.Result, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("parsing doc: %v", err)
	}
	var rows []map[string]interface{}
	doc.Find("#form-product > div > table > tbody > tr").Each(func(_ int, s *goquery.Selection) {
		// Check if no results.
		if strings.TrimSpace(s.Find("td:nth-child(1)").Text()) == MessageNoResults {
			log.Errorln("No results found for current query!")
			return
		}
		rows = append(rows, map[string]interface{}{
			"model":        strings.TrimSpace(s.Find("td:nth-child(4)").Text()),
			"quantity":     strings.TrimSpace(s.Find("td:nth-child(6) > span").Text()),
			"product_name": strings.TrimSpace(s.Find("td:nth-child(3)").Text()),
			"price":        strings.TrimSpace(s.Find("td:nth-child(5)").Text()),
			"status":       strings.TrimSpace(s.Find("td:nth-child(7)").Text()),
			"product_id":   strings.TrimSpace(s.Find("td:nth-child(1) > input").AttrOr("value", "")),
		})
	})
	tokens := pagesRe.FindStringSubmatch(doc.Find("#form-product + div > div + div").Text())
	offset, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("offset")])
	offsetLimit, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("offset_limit")])
	total, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("total")])
	pages, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("pages")])
	// TODO(nmcapule): Handle other errors.
	return utils.GJSONFrom(map[string]interface{}{
		"code":    0,
		"message": "Success",
		"data": map[string]interface{}{
			"rows":   rows,
			"offset": offset - 1,
			"limit":  offsetLimit - offset + 1,
			"total":  total,
			"pages":  pages,
		},
	}), nil
}

func scrapeSaleOrder(input string) (*gjson.Result, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("parsing doc: %v", err)
	}
	var rows []map[string]interface{}
	doc.Find(`div[id^="collapse_products_"] > div > table > tbody > tr`).Each(func(_ int, s *goquery.Selection) {
		model := strings.TrimSpace(s.Find("td:nth-child(3)").Text())
		if model == "" {
			return
		}
		rows = append(rows, map[string]interface{}{
			"model":    model,
			"quantity": strings.TrimSpace(s.Find("td:nth-child(4)").Text()),
		})
	})
	tokens := pagesRe.FindStringSubmatch(doc.Find("#form-order + div > div + div").Text())
	offset, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("offset")])
	offsetLimit, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("offset_limit")])
	total, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("total")])
	pages, _ := strconv.Atoi(tokens[pagesRe.SubexpIndex("pages")])
	// TODO(nmcapule): Handle other errors.
	return utils.GJSONFrom(map[string]interface{}{
		"code":    0,
		"message": "Success",
		"data": map[string]interface{}{
			"rows":   rows,
			"offset": offset - 1,
			"limit":  offsetLimit - offset + 1,
			"total":  total,
			"pages":  pages,
		},
	}), nil
}
