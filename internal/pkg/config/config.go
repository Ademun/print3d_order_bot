package config

type Config struct {
	FileService FileServiceCfg `yaml:"file_service"`
}

type FileServiceCfg struct {
	dirPath             string   `yaml:"dir_path"`
	appendModeFilenames []string `yaml:"append_mode_filenames"`
}
