package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type Config map[string]string
type Configs map[string]Config

func (c *Configs) Merge(tmp *Configs) error {
	for k, v := range *tmp {
		if vv, ok := (*c)[k]; ok {
			for b, s := range v {
				vv[b] = s
			}
		} else {
			(*c)[k] = v
		}
	}
	return nil
}

type PushEvent struct {
	Ref    string `json:"ref"`
	Before string `json:"before"`
	After  string `json:"after"`
	Repo   struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		GitURL   string `json:"git_url"`
		SshURL   string `json:"ssh_url"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

var (
	configFile   = ""
	configString = ""
	secret       = ""
	addr         = ":9090"
	path         = "/deploy"
	certFile     = ""
	keyFile      = ""
	configs      = Configs{}
)

func parse(dat []byte) error {
	cs := Configs{}

	if err := json.Unmarshal(dat, &cs); err != nil {
		c := Config{}

		if err := json.Unmarshal(dat, &c); err != nil {
			return err
		}

		cs = Configs{
			"*": c,
		}
	}

	return configs.Merge(&cs)
}

func init() {
	flag.StringVar(&configFile, "config", configFile, "path to config file")
	flag.StringVar(&configString, "string", configString, "config given as a string")
	flag.StringVar(&secret, "secret", secret, "secret")
	flag.StringVar(&addr, "addr", addr, "network address to listen on")
	flag.StringVar(&path, "path", path, "path")
	flag.StringVar(&certFile, "certFile", certFile, "path to cert file")
	flag.StringVar(&keyFile, "keyFile", keyFile, "path to key file")

	flag.Parse()

	if len(configString) > 0 {
		if err := parse([]byte(configString)); err != nil {
			log.Fatalf("error parsing config string: %v\n", err)
		}
	}

	if len(configFile) > 0 {
		dat, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Fatalf("error reading config file: %v\n", err)
		}

		if err := parse(dat); err != nil {
			log.Fatalf("error parsing config file: %v\n", err)
		}
	}

	if len(configs) == 0 {
		log.Fatalf("no config given\n")
	}
}

func main() {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		evName := r.Header.Get("X-Github-Event")
		if evName != "push" {
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if secret != "" {
			ok := false
			for _, sig := range strings.Fields(r.Header.Get("X-Hub-Signature")) {
				if !strings.HasPrefix(sig, "sha1=") {
					continue
				}
				sig = strings.TrimPrefix(sig, "sha1=")
				mac := hmac.New(sha1.New, []byte(secret))
				mac.Write(body)
				if sig == hex.EncodeToString(mac.Sum(nil)) {
					ok = true
					break
				}
			}
			if !ok {
				log.Printf("Ignoring '%s' event with incorrect signature", evName)
				return
			}
		}

		ev := PushEvent{}
		err = json.Unmarshal(body, &ev)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if config, ok := configs[ev.Repo.FullName]; ok {
			ref := strings.TrimPrefix(ev.Ref, "refs/heads/")

			log.Printf("got a push event: %s, %s, %s\n", ev.Repo.FullName, ref, ev.After)

			if script, ok := config[ref]; ok {
				cmd := exec.Command(script, ev.Repo.SshURL, ref, ev.After)

				// we can also set working dir...
				// cmd.Dir = ...
				var out bytes.Buffer

				cmd.Stdout = &out
				cmd.Stderr = &out

				if err := cmd.Start(); err != nil {
					log.Printf("failed to run `%s` script with error: %v\n", script, err)
					return
				}

				if err := cmd.Wait(); err != nil {
					log.Printf("`%s` script failed with error: %v\n", script, err)
					log.Printf("`````````````\n%s\n`````````````\n", out)
				}
			}
		}
	})

	if certFile != "" && keyFile != "" {
		log.Printf("[https] listening on %s\n", addr)
		if err := http.ListenAndServeTLS(addr, certFile, keyFile, nil); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("[http] listening on %s\n", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal(err)
		}
	}
}
