package config

type Config struct {
	FileService FileServiceCfg `yaml:"file_service"`
}

type FileServiceCfg struct {
	DirPath             string   `yaml:"dir_path"`
	AppendModeFilenames []string `yaml:"append_mode_filenames"`
}
