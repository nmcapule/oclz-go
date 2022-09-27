package opencart

import (
	"time"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/utils/scheduler"

	log "github.com/sirupsen/logrus"
)

func (c *Client) Daemon() models.Daemon {
	return c
}

func (c *Client) Start() error {
	scheduler.Loop(func(quit chan struct{}) {
		log.WithFields(log.Fields{
			"tenant": c.Name,
		}).Infoln("Collecting recent sale orders...")

		// log.Fatalln(c.loadSaleOrderPages(url.Values{
		// 	"filter_date_modified": []string{"2022-09-13"},
		// }))
	}, scheduler.LoopConfig{RetryWait: 5 * time.Second})
	return nil
}
