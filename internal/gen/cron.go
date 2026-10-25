package gen

import "github.com/robfig/cron/v3"

var cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func cronSpecCheck(spec string) (cron.Schedule, error) { return cronParser.Parse(spec) }
