package crypto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/sha3"
)

var blsLock sync.Mutex

func Sha3256(publicKeyBytes []byte) string {
	sha3256 := sha3.New256()
	sha3256.Write(publicKeyBytes)
	var buffer []byte
	clientId := hex.EncodeToString(sha3256.Sum(buffer))
	return clientId
}

func GenerateMnemonic(t *testing.T) string {
	entropy, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropy)
	t.Logf("Generated mnemonic [%s]", mnemonic)

	return mnemonic
}

func GenerateKeys(t *testing.T, mnemonic string) model.KeyPair {
	blsLock.Lock()
	defer blsLock.Unlock()

	_ = bls.Init(bls.CurveFp254BNb)
	seed := bip39.NewSeed(mnemonic, "0chain-client-split-key")
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

func Hash(request *model.Transaction) {
	var transactionDataHash = Sha3256([]byte(request.TransactionData))

	var hashData = blankIfNil(request.CreationDate) + ":" +
		blankIfNil(request.ClientId) + ":" +
		blankIfNil(request.ToClientId) + ":" +
		blankIfNil(request.TransactionValue) + ":" +
		transactionDataHash

	request.Hash = Sha3256([]byte(hashData))
}

func Sign(request *model.Transaction, pair model.KeyPair) {
	hashToSign, _ := hex.DecodeString(request.Hash)
	request.Signature = pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
}

func blankIfNil(obj interface{}) string {
	if obj == nil {
		return ""
	}
	return fmt.Sprintf("%v", obj)
}
