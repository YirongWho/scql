// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/pingcap/errors"

	. "github.com/secretflow/scql/pkg/parser/format"
	"github.com/secretflow/scql/pkg/parser/terror"
)

// UserIdentity represents username and hostname.
type UserIdentity struct {
	Username     string
	Hostname     string
	CurrentUser  bool
	AuthUsername string // Username matched in privileges system
	AuthHostname string // Match in privs system (i.e. could be a wildcard)
}

// Restore implements Node interface.
func (user *UserIdentity) Restore(ctx *RestoreCtx) error {
	if user.CurrentUser {
		ctx.WriteKeyWord("CURRENT_USER")
	} else {
		ctx.WriteName(user.Username)
		if user.Hostname != "" {
			ctx.WritePlain("@")
			ctx.WriteName(user.Hostname)
		}
	}
	return nil
}

// String converts UserIdentity to the format user@host.
func (user *UserIdentity) String() string {
	// TODO: Escape username and hostname.
	if user == nil {
		return ""
	}
	return fmt.Sprintf("%s@%s", user.Username, user.Hostname)
}

// AuthIdentityString returns matched identity in user@host format
func (user *UserIdentity) AuthIdentityString() string {
	// TODO: Escape username and hostname.
	return fmt.Sprintf("%s@%s", user.AuthUsername, user.AuthHostname)
}

type RoleIdentity struct {
	Username string
	Hostname string
}

func (role *RoleIdentity) Restore(ctx *RestoreCtx) error {
	ctx.WriteName(role.Username)
	if role.Hostname != "" {
		ctx.WritePlain("@")
		ctx.WriteName(role.Hostname)
	}
	return nil
}

// String converts UserIdentity to the format user@host.
func (role *RoleIdentity) String() string {
	// TODO: Escape username and hostname.
	return fmt.Sprintf("`%s`@`%s`", role.Username, role.Hostname)
}

// CheckScrambledPassword check scrambled password received from client.
// The new authentication is performed in following manner:
//
//	SERVER:  public_seed=create_random_string()
//	         send(public_seed)
//	CLIENT:  recv(public_seed)
//	         hash_stage1=sha1("password")
//	         hash_stage2=sha1(hash_stage1)
//	         reply=xor(hash_stage1, sha1(public_seed,hash_stage2)
//	         // this three steps are done in scramble()
//	         send(reply)
//	SERVER:  recv(reply)
//	         hash_stage1=xor(reply, sha1(public_seed,hash_stage2))
//	         candidate_hash2=sha1(hash_stage1)
//	         check(candidate_hash2==hash_stage2)
//	         // this three steps are done in check_scramble()
func CheckScrambledPassword(salt, hpwd, auth []byte) bool {
	crypt := sha256.New()
	_, err := crypt.Write(salt)
	terror.Log(errors.Trace(err))
	_, err = crypt.Write(hpwd)
	terror.Log(errors.Trace(err))
	hash := crypt.Sum(nil)
	// token = scrambleHash XOR stage1Hash
	for i := range hash {
		hash[i] ^= auth[i]
	}

	return bytes.Equal(hpwd, Sha256Hash(hash))
}

// Sha256Hash is an util function to calculate sha256 hash.
func Sha256Hash(bs []byte) []byte {
	crypt := sha256.New()
	_, err := crypt.Write(bs)
	terror.Log(errors.Trace(err))
	return crypt.Sum(nil)
}

// EncodePassword converts plaintext password to hashed hex string.
func EncodePassword(pwd string) string {
	if len(pwd) == 0 {
		return ""
	}
	hash1 := Sha256Hash([]byte(pwd))
	hash2 := Sha256Hash(hash1)

	return fmt.Sprintf("*%X", hash2)
}

// DecodePassword converts hex string password without prefix '*' to byte array.
func DecodePassword(pwd string) ([]byte, error) {
	x, err := hex.DecodeString(pwd[1:])
	if err != nil {
		return nil, errors.Trace(err)
	}
	return x, nil
}
