// Encoding: UTF-8
//
// AWS Credential Cachier
//
// Copyright © 2021 Brian Dwyer - Intelligent Digital Services
//

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gofrs/flock"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/hashstructure"
	"github.com/sdomino/scribble"
)

type AwsCredential struct {
	aws.Credentials
	Expiration string
}

type AwsProcessCredential struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string
	SessionToken    string `json:",omitempty"`
	Expiration      string `json:",omitempty"`
	Version         int
}

func (c *AwsCredential) ToJson() (jsonBytes []byte) {
	jsonBytes, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return
}

func (c *AwsCredential) ToProcessJson() (jsonBytes []byte) {
	cred := AwsProcessCredential{
		AccessKeyID:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		SessionToken:    c.SessionToken,
		Expiration:      c.Expiration,
		Version:         1,
	}
	jsonBytes, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return
}

var disableSharedConfig bool

func init() {
	if flag.Lookup("disable-shared-config") == nil {
		flag.BoolVar(&disableSharedConfig, "disable-shared-config", false, "Disable Shared Configuration (force use of EC2/ECS metadata, ignore AWS_PROFILE, etc.)")
	}
}

func main() {
	// Parse Flags
	flag.Parse()

	if versionFlag {
		showVersion()
		os.Exit(0)
	}

	// Calculate Request Hash (Args + AWS Env Vars)
	req := append([]string{}, os.Args[1:]...)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "AWS") {
			req = append(req, env)
		}
	}
	hash, err := hashstructure.Hash(req, nil)
	if err != nil {
		log.Fatal(err)
	}
	csum := strconv.FormatUint(hash, 10)

	// Loop Detection
	if callingCsum := os.Getenv("_AWS_CRED_CACHIER_CSUM"); callingCsum == csum {
		log.Fatal("Loop detected! Called recursively by PID: ", os.Getenv("_AWS_CRED_CACHIER_PID"))
	}
	os.Setenv("_AWS_CRED_CACHIER_CSUM", csum)
	os.Setenv("_AWS_CRED_CACHIER_PID", strconv.Itoa(os.Getpid()))

	// Attempt to Read Credentials
	if ok := Read(csum); ok {
		return
	}

	f := flock.New(filepath.Join(dbPath, ".lock"))
	rand.Seed(time.Now().Unix() + int64(os.Getpid()))
	for {
		f.Lock()
		if f.Locked() {
			break
		}
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		// Retry Reading Credentials
		Read(csum)
	}
	defer f.Unlock()

	var cfg aws.Config
	if disableSharedConfig {
		cfg, err = config.LoadDefaultConfig(context.Background(), config.WithSharedConfigFiles([]string{}), config.WithSharedCredentialsFiles([]string{}))
	} else {
		cfg, err = config.LoadDefaultConfig(context.Background())
	}
	if err != nil {
		panic(err)
	}

	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	var expiresAt time.Time
	if creds.CanExpire {
		expiresAt = creds.Expires
	} else {
		expiresAt = time.Now().Add(time.Minute * 5)
	}

	cred := AwsCredential{creds, expiresAt.Format(time.RFC3339)}

	// Write to Cache
	if err := Db().Write("cdb", csum, cred); err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(cred.ToProcessJson()))
}

var dbPath string

func Db() (db *scribble.Driver) {

	if dbPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}
		dbPath = filepath.Join(home, ".aws-cred-cachier")
	}
	db, err := scribble.New(dbPath, nil)
	if err != nil {
		log.Fatal(err)
	}

	return
}

// Read cached credentials and emit them if valid
func Read(request_hash string) (valid bool) {
	cred := AwsCredential{}
	if err := Db().Read("cdb", request_hash, &cred); err == nil {
		if cred.Expiration != "" {
			expires, err := time.Parse(time.RFC3339, cred.Expiration)
			if err != nil {
				log.Fatal(err)
			}
			if expires.After(time.Now().Add(time.Minute * 1)) {
				valid = true
				fmt.Println(string(cred.ToProcessJson()))
			}
		}
	}
	return
}
