// Encoding: UTF-8
//
// AWS Credential Cachier
//
// Copyright © 2020 Brian Dwyer - Intelligent Digital Services
//

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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

	// AWS Session
	sess_opts := session.Options{
		// Config:            *aws.NewConfig().WithRegion("us-east-1"),
		SharedConfigState: session.SharedConfigEnable,
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

func Db() (db *scribble.Driver) {
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := filepath.Join(home, ".aws-cred-cachier")
	if db, err = scribble.New(dbPath, nil); err != nil {
		log.Fatal(err)
	}

	return
}
