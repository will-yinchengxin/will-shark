package config

import (
	"github.com/spf13/viper"
	"willshark/consts"
)

type Setting struct {
	vp *viper.Viper
}

func NewSetting(env string) (*Setting, error) {
	vp := viper.New()
	vp.AddConfigPath(consts.ConfigPath)
	vp.SetConfigName(env + consts.ConfigName)
	vp.SetConfigType(consts.ConfigType)
	err := vp.ReadInConfig()
	if err != nil {
		return nil, err
	}
	return &Setting{vp}, nil
}

func (s *Setting) ReadSection(k string, v interface{}) error {
	err := s.vp.UnmarshalKey(k, v)
	if err != nil {
		return err
	}
	return nil
}
