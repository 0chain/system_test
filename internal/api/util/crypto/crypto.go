package crypto

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/stretchr/testify/require"
	"github.com/tyler-smith/go-bip39" //nolint
	"golang.org/x/crypto/sha3"
	"log"
	"sync"
	"testing"
)

var blsLock sync.Mutex

func init() {
	blsLock.Lock()
	defer blsLock.Unlock()

	err := bls.Init(bls.CurveFp254BNb)

	if err != nil {
		panic(err)
	}
}

func GenerateMnemonics(t *testing.T) string {
	entropy, err := bip39.NewEntropy(256) //nolint
	if err != nil {
		log.Fatalln(err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy) //nolint
	if err != nil {
		log.Fatalln(err)
	}
	t.Logf("Generated mnemonic [%s]", mnemonic)

	return mnemonic
}

func GenerateKeys(t *testing.T, mnemonics string) *model.KeyPair {
	defer handlePanic(t)
	blsLock.Lock()
	defer func() {
		blsLock.Unlock()
		bls.SetRandFunc(nil)
	}()

	seed := bip39.NewSeed(mnemonics, "0chain-client-split-key") //nolint
	random := bytes.NewReader(seed)
	bls.SetRandFunc(random)

	var secretKey bls.SecretKey
	secretKey.SetByCSPRNG()

	publicKey := secretKey.GetPublicKey()
	secretKeyHex := secretKey.SerializeToHexStr()
	publicKeyHex := publicKey.SerializeToHexStr()

	t.Logf("Generated public key [%s] and secret key [%s]", publicKeyHex, secretKeyHex)
	bls.SetRandFunc(nil)

	return &model.KeyPair{PublicKey: *publicKey, PrivateKey: secretKey}
}

func Sha3256(src []byte) string {
	sha3256 := sha3.New256()
	sha3256.Write(src)
	var buffer []byte
	return hex.EncodeToString(sha3256.Sum(buffer))
}

func SignTransaction(t *testing.T, request *model.TransactionPutRequest, pair *model.KeyPair) {
	defer handlePanic(t)
	blsLock.Lock()
	defer blsLock.Unlock()

	hashToSign, err := hex.DecodeString(request.Hash)
	require.NoError(t, err, "Error on hash")

	request.Signature = pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
}

func HashTransaction(request *model.TransactionEntity) {
	var hashData = blankIfNil(request.CreationDate) + ":" +
		blankIfNil(request.TransactionNonce) + ":" +
		blankIfNil(request.ClientId) + ":" +
		blankIfNil(request.ToClientId) + ":" +
		blankIfNil(request.TransactionValue) + ":" +
		Sha3256([]byte(request.TransactionData))

	request.Hash = Sha3256([]byte(hashData))
}

func blankIfNil(obj interface{}) string {
	if obj == nil {
		return ""
	}
	return fmt.Sprintf("%v", obj)
}

func handlePanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Errorf("panic occurred: ", err)
	}
}
