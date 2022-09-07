package crypto

import (
	"bytes"
	"crypto/sha1"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/lithammer/shortuuid/v3"
	"github.com/tyler-smith/go-bip39" //nolint
	"golang.org/x/crypto/sha3"
)

var blsLock sync.Mutex

const BLS0Chain = "bls0chain"

func Sha3256(publicKeyBytes []byte) string {
	sha3256 := sha3.New256()
	sha3256.Write(publicKeyBytes)
	var buffer []byte
	clientId := hex.EncodeToString(sha3256.Sum(buffer))
	return clientId
}

func GenerateMnemonic(t *testing.T) string {
	entropy, _ := bip39.NewEntropy(256)       //nolint
	mnemonic, _ := bip39.NewMnemonic(entropy) //nolint
	t.Logf("Generated mnemonic [%s]", mnemonic)

	return mnemonic
}

func GenerateKeys(t *testing.T, mnemonic string) model.KeyPair {
	blsLock.Lock()
	defer blsLock.Unlock()

	_ = bls.Init(bls.CurveFp254BNb)
	seed := bip39.NewSeed(mnemonic, "0chain-client-split-key") //nolint
	random := bytes.NewReader(seed)
	bls.SetRandFunc(random)

	var secretKey bls.SecretKey
	secretKey.SetByCSPRNG()

	publicKey := secretKey.GetPublicKey()
	secretKeyHex := secretKey.SerializeToHexStr()
	publicKeyHex := publicKey.SerializeToHexStr()

	t.Logf("Generated public key [%s] and secret key [%s]", publicKeyHex, secretKeyHex)
	bls.SetRandFunc(nil)

	return model.KeyPair{PublicKey: *publicKey, PrivateKey: secretKey}
}

func NewConnectionID() string {
	return shortuuid.New()
}

func HashOfFile(src *os.File) (string, error) {
	h := sha1.New()
	if _, err := io.Copy(h, src); err != nil {
		return "", err
	}
	src.Seek(0, io.SeekStart)

	return hex.EncodeToString(h.Sum(nil)), nil
}

func Hash(src string) string { return Sha3256([]byte(src)) }

func Sign(hash string, pair model.KeyPair) string {
	return pair.PrivateKey.Sign(string(hash)).SerializeToHexStr()
}

func HashTransaction(request *model.Transaction) {
	var hashData = blankIfNil(request.CreationDate) + ":" +
		blankIfNil(request.TransactionNonce) + ":" +
		blankIfNil(request.ClientId) + ":" +
		blankIfNil(request.ToClientId) + ":" +
		blankIfNil(request.TransactionValue) + ":" +
		Sha3256([]byte(request.TransactionData))

	request.Hash = Sha3256([]byte(hashData))
}

func SignTransaction(request *model.Transaction, pair model.KeyPair) {
	hashToSign, _ := hex.DecodeString(request.Hash)
	request.Signature = pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
}

func blankIfNil(obj interface{}) string {
	if obj == nil {
		return ""
	}
	return fmt.Sprintf("%v", obj)
}
