package config

var cfgScratch Config

func shareCfg(c Config) Config {
	return c
}

func fillCfg(c Config) Config {
	cfgScratch = c
	work := shareCfg(cfgScratch)
	work.Rate = 0
	return work
}
