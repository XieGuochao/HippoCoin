package host

import (
	cryptoRand "crypto/rand"

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
	_ "golang.org/x/crypto/ripemd160" // for sha256
)

// Key ...
type Key struct {
	key       *ecdsa.PrivateKey
	publicKey *ecdsa.PublicKey
	curve     elliptic.Curve
}

// New ...
func (k *Key) New(curve elliptic.Curve) {
	k.curve = curve
}

// Key ...
func (k *Key) Key() *ecdsa.PrivateKey {
	return k.key
}

// GenerateKey ...
func (k *Key) GenerateKey() {
	k.key = GenerateKey(k.curve)
	k.publicKey = &k.key.PublicKey
}

// SaveKey ...
func (k *Key) SaveKey(privateKeyFile string) {
	SaveKey(k.key, privateKeyFile)
}

// ToAddress ...
// func (k *Key) ToAddress() string {
// 	return publicKeyToAddress(k.key.PublicKey)
// }
func (k *Key) ToAddress() string {
	return publicKeyToString(*k.publicKey)
}

// LoadAddress ...
func (k *Key) LoadAddress(address string, curve elliptic.Curve) {
	k.publicKey = stringToPublicKey(address, curve)
	k.curve = curve
	k.key = nil
}

// LoadPublicKeyString ...
func (k *Key) LoadPublicKeyString(publicKeyString string, curve elliptic.Curve) {
	k.publicKey = stringToPublicKey(publicKeyString, curve)
	k.curve = curve
}

// ValidateAddress ...
func (k *Key) ValidateAddress(address string) bool {
	return validatePublicKeyAddress(*k.publicKey, address)
}

// Sign ...
func (k *Key) Sign(input []byte) ([]byte, error) {
	return k.key.Sign(cryptoRand.Reader, input, nil)
}

// SignString ...
func (k *Key) SignString(input []byte) (string, error) {
	b, e := k.Sign(input)
	if e != nil {
		return "", e
	}
	return ByteToHexString(b), nil
}

// CheckSign ...
func (k *Key) CheckSign(hash []byte, sig []byte) bool {
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
	return ecdsa.Verify(k.publicKey, hash, r, s)
}

// CheckSignString ...
func (k *Key) CheckSignString(hash string, sig string) bool {
	h, err := StringToByte(hash)
	if err != nil {
		debugLogger.Debug("check sign string failed:", err)
		return false
	}
	s, err := StringToByte(sig)
	if err != nil {
		debugLogger.Debug("check sign string failed:", err)
		return false
	}
	return k.CheckSign(h, s)
}

// PrivateKeyString ...
func (k *Key) PrivateKeyString() string {
	return KeyToHexString(k.key)
}

// LoadPrivateKeyString ...
func (k *Key) LoadPrivateKeyString(priString string, curve elliptic.Curve) error {
	key, err := HexStringToKey(priString)
	if err != nil {
		return err
	}
	k.key = key
	k.publicKey = &key.PublicKey
	return nil
}

// ========================================================

// GenerateKey ...
func GenerateKey(curve elliptic.Curve) *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Println(err)
		return nil
	}
	return key
}

// SaveKey ...
func SaveKey(key *ecdsa.PrivateKey, privateKeyFile string) {
	privateKeyBytes := keyToByte(key)
	infoLogger.Info("Create one key pair:", privateKeyFile)

	ioutil.WriteFile(privateKeyFile, privateKeyBytes, 0644)
}

func publicKeyToAddress(publicKey ecdsa.PublicKey) string {
	sha256 := crypto.SHA256.New()
	r := crypto.RIPEMD160.New()
	sha256.Write([]byte(publicKeyToString(publicKey)))
	r.Write(sha256.Sum(nil))
	rSum := r.Sum(nil)
	checkSum := rSum[:4]
	return base64.StdEncoding.EncodeToString(append(rSum, checkSum...))
}

// keyPairToByte ...
func keyToByte(privateKey *ecdsa.PrivateKey) (priByte []byte) {
	if privateKey == nil {
		return []byte{}
	}
	priByte, _ = x509.MarshalECPrivateKey(privateKey)
	return
}

// KeyToHexString ...
func KeyToHexString(privateKey *ecdsa.PrivateKey) (priString string) {
	priByte := keyToByte(privateKey)
	return ByteToHexString(priByte)
}

// byteToKey ...
func byteToKey(priByte []byte) (privateKey *ecdsa.PrivateKey, err error) {
	privateKey, err = x509.ParseECPrivateKey(priByte)
	return
}

// HexStringToKey ...
func HexStringToKey(priString string) (privateKey *ecdsa.PrivateKey, err error) {
	var priBytes []byte
	priBytes, err = StringToByte(priString)
	if err != nil {
		return
	}
	return byteToKey(priBytes)
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

// publicKeyToString ...
// Convert publicKey into hexadecimal strings
func publicKeyToString(publicKey ecdsa.PublicKey) string {
	return publicKey.X.Text(16) + "|" + publicKey.Y.Text(16)
}

// validate if the publicKey and the address matched
func validatePublicKeyAddress(publicKey ecdsa.PublicKey, address string) bool {
	return publicKeyToAddress(publicKey) == address
}
