// Encoding: UTF-8
//
// AWS Credential Cachier
//
// Copyright © 2020 Brian Dwyer - Intelligent Digital Services
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

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gofrs/flock"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/hashstructure"
	"github.com/sdomino/scribble"
)

type AwsCredential struct {
	credentials.Value
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

func main() {
	// Loop Detection
	if callingPid := os.Getenv("_AWS_CRED_CACHIER_PID"); callingPid != "" {
		log.Fatal("Loop detected! Called recursively by PID: ", callingPid)
	}
	os.Setenv("_AWS_CRED_CACHIER_PID", string(os.Getpid()))

	disableSharedConfig := flag.Bool("disable-shared-config", false, "Disable Shared Configuration (force use of EC2/ECS metadata, ignore AWS_PROFILE, etc.)")
	flag.Parse()
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

	cred := AwsCredential{}
	if err := Db().Read("cdb", csum, &cred); err == nil {
		if cred.Expiration != "" {
			expires, err := time.Parse(time.RFC3339, cred.Expiration)
			if err != nil {
				log.Fatal(err)
			}
			if expires.After(time.Now().Add(time.Minute * 1)) {
				fmt.Println(string(cred.ToProcessJson()))
				return
			}
		}
	}

	f := flock.New(filepath.Join(dbPath, ".lock"))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rand.Seed(time.Now().Unix() + int64(os.Getpid()))
	if _, err := f.TryLockContext(lockCtx, time.Duration(rand.Intn(250)+500)*time.Millisecond); err != nil {
		log.Fatal(err)
	}
	defer f.Unlock()

	// AWS Session
	sess_opts := session.Options{
		// Config:            *aws.NewConfig().WithRegion("us-east-1"),
		SharedConfigState: session.SharedConfigEnable,
	}
	if *disableSharedConfig {
		sess_opts.SharedConfigState = session.SharedConfigDisable
	}

	sess := session.Must(session.NewSessionWithOptions(sess_opts))

	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		log.Fatal(err)
	}

	expiresAt, err := sess.Config.Credentials.ExpiresAt()
	if err != nil {
		expiresAt = time.Now().Add(time.Minute * 5)
	}

	cred = AwsCredential{creds, expiresAt.Format(time.RFC3339)}

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
