package cron

import (
	"fmt"
)

func (j *Jobs) oneCronJob() {
	fmt.Println("this is job one")
	// you own job
}
