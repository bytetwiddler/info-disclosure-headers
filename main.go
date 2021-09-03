package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

// Config struct object to read in yaml config data
type Config struct {
	Sites   []string `yaml:"sites"`
	Methods []string `yaml:"methods"`
}

func worker(method string, uri string, printAll bool, allHeaders bool, wg *sync.WaitGroup ) {
	defer wg.Done()
	client := &http.Client{}
	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)


	if printAll {
		fmt.Printf("%v|%v|%v|%v|%v|%v\n",
			uri,
			method,
			resp.StatusCode,
			resp.Header.Get("Server"),
			resp.Header.Get("X-Powered-By"),
			resp.Header.Get("Max-Forwards"))
	} else {
		if !allHeaders && resp.StatusCode < 400 {
			fmt.Printf("%v|%v|%v|%v|%v|%v\n",
				uri,
				method,
				resp.StatusCode,
				resp.Header.Get("Server"),
				resp.Header.Get("X-Powered-By"),
				resp.Header.Get("Max-Forwards"))
		}
	}
	if allHeaders {
		fmt.Printf("%v - %v - Status Code:%v\n", uri, method, resp.StatusCode)
		headers := make(map[string]interface{})
		for k, v := range resp.Header {
			headers[strings.ToLower(k)] = string(v[0])
			fmt.Printf("%v:%v\n", k, v[0])
		}
		fmt.Println()
	}
}

// NewConfig - function to read in yaml config file
func NewConfig(configPath string) (*Config, error) {

	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	d := yaml.NewDecoder(file)

	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

// ValidateConfigPath - function to validate path to config.yml
func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

// ParseFlags - function to parse command line flags
func ParseFlags() (string, bool, bool, error) {
	var configPath string
	var printAll  bool
	var allHeaders bool
	flag.StringVar(&configPath, "config", "./config.yml", "path to config file")
	flag.BoolVar(&printAll, "all", false, "all status codes")
	flag.BoolVar(&allHeaders, "headers", false, "all headers for each method")
	flag.Parse()

	if err := ValidateConfigPath(configPath); err != nil {
		return "", printAll, allHeaders, err
	}
	return configPath, printAll, allHeaders, nil
}

func main() {
	cfgPath, printAll, allHeaders, err := ParseFlags()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(99)
	}

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(99)
	}

	if allHeaders == false {
		fmt.Println("Site|Method|StatusCode|Server Header|X-Powered-By Header|Max-Forwards Header")
	}

	var wg sync.WaitGroup
	wg.Add(len(cfg.Methods) * len(cfg.Sites))
	for _, uri := range cfg.Sites {
		for _, method := range cfg.Methods {
			go worker(method, uri, printAll, allHeaders, &wg)
		}
	}
	wg.Wait()
}
