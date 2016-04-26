// Copyright 2016 Paul Stuart. All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file.

package snmputil

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/soniah/gosnmp"
)

const (
	defaultPort = 161
)

var (
	authProto = map[string]gosnmp.SnmpV3AuthProtocol{
		"NoAuth": gosnmp.NoAuth,
		"MD5":    gosnmp.MD5,
		"SHA":    gosnmp.SHA,
	}
	privacy = map[string]gosnmp.SnmpV3PrivProtocol{
		"NoPriv": gosnmp.NoPriv,
		"DES":    gosnmp.DES,
		"AES":    gosnmp.AES,
	}

	ErrBadUser     = fmt.Errorf("missing snmp v3 username")
	ErrBadPassword = fmt.Errorf("missing snmp v3 user password")
	ErrBadProtocol = fmt.Errorf("invalid snmp v3 auth protocol")
	ErrLevel       = fmt.Errorf("invalid snmp v3 security level")
	ErrPrivacy     = fmt.Errorf("missing snmp v3 privacy password")
	ErrVersion     = fmt.Errorf("invalid snmp version")
)

type Profile struct {
	Host, Community, Version string
	Port, Timeout, Retries   int
	// for SNMP v3
	SecLevel, AuthUser, AuthPass, AuthProto, PrivProto, PrivPass string
}

// NewClient returns an snmp client that has connected to an snmp agent
func NewClient(p Profile) (*gosnmp.GoSNMP, error) {

	var ok bool
	var aProto gosnmp.SnmpV3AuthProtocol
	var pProto gosnmp.SnmpV3PrivProtocol
	var msgFlags gosnmp.SnmpV3MsgFlags

	authCheck := func() error {
		if len(p.AuthPass) < 1 {
			log.Printf("Error no SNMPv3 password for host %s", p.Host)
			return ErrBadPassword
		}
		if aProto, ok = authProto[p.AuthProto]; !ok {
			log.Printf("Error in Auth Protocol %s for host %s", p.AuthProto, p.Host)
			return ErrBadProtocol
		}
		return nil
	}

	v3auth := func() (*gosnmp.UsmSecurityParameters, error) {
		if len(p.AuthUser) < 1 {
			log.Printf("Error username not found in snmpv3 %s in host %s", p.AuthUser, p.Host)
			return nil, ErrBadUser
		}

		switch p.SecLevel {
		case "NoAuthNoPriv":
			msgFlags = gosnmp.NoAuthNoPriv
			return &gosnmp.UsmSecurityParameters{
				UserName:               p.AuthUser,
				AuthenticationProtocol: gosnmp.NoAuth,
				PrivacyProtocol:        gosnmp.NoPriv,
			}, nil
		case "AuthNoPriv":
			msgFlags = gosnmp.AuthNoPriv
			return &gosnmp.UsmSecurityParameters{
				UserName:                 p.AuthUser,
				AuthenticationProtocol:   aProto,
				AuthenticationPassphrase: p.AuthPass,
				PrivacyProtocol:          gosnmp.NoPriv,
			}, authCheck()
		case "AuthPriv":
			msgFlags = gosnmp.AuthPriv
			if len(p.PrivPass) < 1 {
				log.Printf("Error privPass not found in snmpv3 for host %s", p.Host)
				return nil, ErrPrivacy
			}

			if pProto, ok = privacy[p.PrivProto]; !ok {
				log.Printf("Error in Priv Protocol %s for host %s", p.PrivProto, p.Host)
				return nil, ErrBadPassword
			}

			return &gosnmp.UsmSecurityParameters{
				UserName:                 p.AuthUser,
				AuthenticationProtocol:   aProto,
				AuthenticationPassphrase: p.AuthPass,
				PrivacyProtocol:          pProto,
				PrivacyPassphrase:        p.PrivPass,
			}, authCheck()

		default:
			log.Printf("invalid security level %s for host %s", p.SecLevel, p.Host)
			return nil, ErrLevel
		}
	}

	if p.Port == 0 {
		p.Port = defaultPort
	}

	client := &gosnmp.GoSNMP{
		Target:  p.Host,
		Port:    uint16(p.Port),
		Timeout: time.Duration(p.Timeout) * time.Second,
		Retries: p.Retries,
	}

	switch p.Version {
	case "1":
		client.Version = gosnmp.Version1
		client.Community = p.Community
	case "2", "2c":
		client.Version = gosnmp.Version2c
		client.Community = p.Community
	case "3":
		usmParams, err := v3auth()
		if err != nil {
			return nil, err
		}
		client.MsgFlags = msgFlags
		client.SecurityModel = gosnmp.UserSecurityModel
		client.SecurityParameters = usmParams
		client.Version = gosnmp.Version3
	default:
		return nil, ErrVersion
	}

	if Debug {
		client.Logger = log.New(os.Stderr, "", 0)
	}

	return client, client.Connect()
}