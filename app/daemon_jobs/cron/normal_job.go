package cron

import (
	"fmt"
	"time"
	"will/utils"
)

func jobOne(t *time.Ticker) {
	var err error
	defer func() {
		t.Reset(time.Second * 2)
		utils.ErrorLog(err)
	}()
	fmt.Println("this is job one")
	// you own job
}
