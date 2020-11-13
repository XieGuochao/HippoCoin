package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"log"
	"math/big"
	"strings"

	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
	_ "golang.org/x/crypto/ripemd160"
)

// PrivateKey ...
func (client *HippoCoinClient) PrivateKey() *ecdsa.PrivateKey {
	return client.privateKey
}

// PublicKey ...
func (client *HippoCoinClient) PublicKey() *ecdsa.PublicKey {
	return &client.privateKey.PublicKey
}

// GenerateKeyPair ...
func (client *HippoCoinClient) GenerateKeyPair() *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(client.curve, rand.Reader)
	if err != nil {
		log.Println(err)
		return nil
	}
	client.privateKey = key

	privateKeyBytes := keyToByte(key)
	log.Println("Create one key pair:", client.privateKeyFile)
	// log.Println(privateKeyBytes)

	ioutil.WriteFile(client.privateKeyFile, privateKeyBytes, 0644)

	return client.privateKey
}

// LoadPrivateKey ...
func (client *HippoCoinClient) LoadPrivateKey() *ecdsa.PrivateKey {
	dat, err := ioutil.ReadFile(client.privateKeyFile)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	key, err := byteToKey(dat)
	if err != nil {
		log.Println(err)
		return nil
	}
	return key
}

// Address ...
// The address is the public key.
func (client *HippoCoinClient) Address() string {
	return PublicKeyToAddress(&client.privateKey.PublicKey)
}

// PublicKeyToAddress ...
// Translate the publicKey to an address string.
func PublicKeyToAddress(publicKey *ecdsa.PublicKey) string {
	sha256 := crypto.SHA256.New()
	r := crypto.RIPEMD160.New()
	sha256.Write([]byte(publicKeyToString(publicKey)))
	r.Write(sha256.Sum(nil))
	rSum := r.Sum(nil)
	checkSum := rSum[:4]
	return base64.StdEncoding.EncodeToString(append(rSum, checkSum...))
}

// ValidatePublicKeyAddress ...
// Validate if the publicKey and the Address matched.
func ValidatePublicKeyAddress(publicKey *ecdsa.PublicKey, address string) bool {
	return PublicKeyToAddress(publicKey) == address
}

// StringToPublicKey ...
// Based on the current curve, translate the public key from string.
func (client *HippoCoinClient) StringToPublicKey(s string) *ecdsa.PublicKey {
	return stringToPublicKey(s, client.curve)
}

// publicKeyToString ...
// Convert publicKey into hexadecimal strings
func publicKeyToString(publicKey *ecdsa.PublicKey) string {
	return publicKey.X.Text(16) + "|" + publicKey.Y.Text(16)
}

// stringToPublicKey ...
// Convert string into publicKey
func stringToPublicKey(s string, curve elliptic.Curve) *ecdsa.PublicKey {
	newPub := new(ecdsa.PublicKey)
	newPub.X = new(big.Int)
	newPub.Y = new(big.Int)

	newPub.Curve = curve
	splitS := strings.SplitN(s, "|", 2)
	if len(splitS) != 2 {
		return nil
	}
	_, check := newPub.X.SetString(splitS[0], 16)
	if !check {
		return nil
	}
	_, check = newPub.Y.SetString(splitS[1], 16)
	if !check {
		return nil
	}
	return newPub
}

// keyPairToByte ...
func keyToByte(privateKey *ecdsa.PrivateKey) (priByte []byte) {
	priByte, _ = x509.MarshalECPrivateKey(privateKey)
	return
}

// byteToKey ...
func byteToKey(priByte []byte) (privateKey *ecdsa.PrivateKey, err error) {
	privateKey, err = x509.ParseECPrivateKey(priByte)
	return
}

// Verify ...
// The general function to verify the given signature, hash, and publicKey.
func Verify(publicKey *ecdsa.PublicKey, sig []byte, hash []byte, client *HippoCoinClient) bool {
	var (
		r, s  = &big.Int{}, &big.Int{}
		inner cryptobyte.String
	)
	input := cryptobyte.String(sig)
	if !input.ReadASN1(&inner, asn1.SEQUENCE) ||
		!input.Empty() ||
		!inner.ReadASN1Integer(r) ||
		!inner.ReadASN1Integer(s) ||
		!inner.Empty() {
		return false
	}
	return ecdsa.Verify(publicKey, hash, r, s)
}
