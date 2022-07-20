package cliutils

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/errors"

	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/gosdk/core/util"
	"github.com/0chain/gosdk/zboxcore/blockchain"
	"github.com/0chain/gosdk/zboxcore/client"
	"github.com/sirupsen/logrus"
)

var Logger = getLogger()

func RunCommandWithoutRetry(commandString string) ([]string, error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]

	sanitizedArgs := sanitizeArgs(args)
	rawOutput, err := executeCommand(commandName, sanitizedArgs)

	Logger.Debugf("Command [%v] exited with error [%v] and output [%v]", commandString, err, sanitizeOutput(rawOutput))

	return sanitizeOutput(rawOutput), err
}

func RunCommandWithRawOutput(commandString string) ([]string, error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]

	sanitizedArgs := sanitizeArgs(args)
	rawOutput, err := executeCommand(commandName, sanitizedArgs)

	Logger.Debugf("Command [%v] exited with error [%v] and output [%v]", commandString, err, string(rawOutput))

	output := strings.Split(string(rawOutput), "\n")

	return output, err
}

func RunCommand(t *testing.T, commandString string, maxAttempts int, backoff time.Duration) ([]string, error) {
	red := "\033[31m"
	yellow := "\033[33m"
	green := "\033[32m"

	var count int
	for {
		count++
		output, err := RunCommandWithoutRetry(commandString)

		if err == nil {
			if count > 1 {
				t.Logf("%sCommand passed on retry [%v/%v]. Output: [%v]\n", green, count, maxAttempts, strings.Join(output, " -<NEWLINE>- "))
			}
			return output, nil
		} else if count < maxAttempts {
			t.Logf("%sCommand failed on attempt [%v/%v] due to error [%v]. Output: [%v]\n", yellow, count, maxAttempts, err, strings.Join(output, " -<NEWLINE>- "))
			time.Sleep(backoff)
		} else {
			t.Logf("%sCommand failed on final attempt [%v/%v] due to error [%v]. Command String: [%v] Output: [%v]\n", red, count, maxAttempts, err, commandString, strings.Join(output, " -<NEWLINE>- "))

			if err != nil {
				t.Logf("%sThe verbose output for the command is:", red)
				commandString = strings.Replace(commandString, "--silent", "", 1)
				out, _ := RunCommandWithoutRetry(commandString) // Only for logging!
				for _, line := range out {
					t.Logf("%s%s", red, line)
				}
			}

			return output, err
		}
	}
}

func StartCommand(t *testing.T, commandString string, maxAttempts int, backoff time.Duration) (cmd *exec.Cmd, err error) {
	var count int
	for {
		count++
		cmd, err := StartCommandWithoutRetry(commandString)

		if err == nil {
			if count > 1 {
				t.Logf("Command started on retry [%v/%v].", count, maxAttempts)
			}
			return cmd, err
		} else if count < maxAttempts {
			t.Logf("Command failed on attempt [%v/%v] due to error [%v]\n", count, maxAttempts, err)
			t.Logf("Sleeping for backoff duration: %v\n", backoff)
			_ = cmd.Process.Kill()
			time.Sleep(backoff)
		} else {
			t.Logf("Command failed on final attempt [%v/%v] due to error [%v].\n", count, maxAttempts, err)
			_ = cmd.Process.Kill()
			return cmd, err
		}
	}
}

func StartCommandWithoutRetry(commandString string) (cmd *exec.Cmd, err error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]
	sanitizedArgs := sanitizeArgs(args)

	cmd = exec.Command(commandName, sanitizedArgs...)
	Setpgid(cmd)
	err = cmd.Start()

	return cmd, err
}

func RandomAlphaNumericString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret)
}

func Wait(t *testing.T, duration time.Duration) {
	t.Logf("Waiting %s...", duration)
	time.Sleep(duration)
}

func sanitizeOutput(rawOutput []byte) []string {
	output := strings.Split(string(rawOutput), "\n")
	var sanitizedOutput []string

	for _, lineOfOutput := range output {
		uniqueOutput := strings.Join(unique(strings.Split(lineOfOutput, "\r")), " ")
		trimmedOutput := strings.TrimSpace(uniqueOutput)
		if trimmedOutput != "" {
			sanitizedOutput = append(sanitizedOutput, trimmedOutput)
		}
	}

	return unique(sanitizedOutput)
}

func unique(slice []string) []string {
	var uniqueOutput []string
	existingOutput := make(map[string]bool)

	for _, element := range slice {
		trimmedElement := strings.TrimSpace(element)
		if _, existing := existingOutput[trimmedElement]; !existing {
			existingOutput[trimmedElement] = true
			uniqueOutput = append(uniqueOutput, trimmedElement)
		}
	}

	return uniqueOutput
}

func executeCommand(commandName string, args []string) ([]byte, error) {
	cmd := exec.Command(commandName, args...)
	rawOutput, err := cmd.CombinedOutput()

	return rawOutput, err
}

func sanitizeArgs(args []string) []string {
	var sanitizedArgs []string
	for _, arg := range args {
		sanitizedArgs = append(sanitizedArgs, strings.ReplaceAll(arg, "\"", ""))
	}

	return sanitizedArgs
}

func parseCommand(command string) []string {
	commandArgSplitter := regexp.MustCompile(`[^\s"]+|"([^"]*)"`)
	fullCommand := commandArgSplitter.FindAllString(command, -1)

	return fullCommand
}

func getLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout

	logger.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
	})

	if strings.EqualFold(strings.TrimSpace(os.Getenv("DEBUG")), "true") {
		logger.SetLevel(logrus.DebugLevel)
	}

	return logger
}

func Contains(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

const SC_REST_API_URL = "v1/screst/"
const REGISTER_CLIENT = "v1/client/put"

const MAX_RETRIES = 5
const SLEEP_BETWEEN_RETRIES = 5

// In percentage
const consensusThresh = float32(25.0)

type SCRestAPIHandler func(response map[string][]byte, numSharders int, err error)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var Client HttpClient

const (
	ALLOCATION_ENDPOINT      = "/allocation"
	UPLOAD_ENDPOINT          = "/v1/file/upload/"
	RENAME_ENDPOINT          = "/v1/file/rename/"
	COPY_ENDPOINT            = "/v1/file/copy/"
	LIST_ENDPOINT            = "/v1/file/list/"
	REFERENCE_ENDPOINT       = "/v1/file/referencepath/"
	CONNECTION_ENDPOINT      = "/v1/connection/details/"
	COMMIT_ENDPOINT          = "/v1/connection/commit/"
	DOWNLOAD_ENDPOINT        = "/v1/file/download/"
	LATEST_READ_MARKER       = "/v1/readmarker/latest"
	FILE_META_ENDPOINT       = "/v1/file/meta/"
	FILE_STATS_ENDPOINT      = "/v1/file/stats/"
	OBJECT_TREE_ENDPOINT     = "/v1/file/objecttree/"
	REFS_ENDPOINT            = "/v1/file/refs/"
	COMMIT_META_TXN_ENDPOINT = "/v1/file/commitmetatxn/"
	COLLABORATOR_ENDPOINT    = "/v1/file/collaborator/"
	CALCULATE_HASH_ENDPOINT  = "/v1/file/calculatehash/"
	SHARE_ENDPOINT           = "/v1/marketplace/shareinfo/"
	DIR_ENDPOINT             = "/v1/dir/"

	// CLIENT_SIGNATURE_HEADER represents http request header contains signature.
	CLIENT_SIGNATURE_HEADER = "X-App-Client-Signature"
)

func getEnvAny(names ...string) string {
	for _, n := range names {
		if val := os.Getenv(n); val != "" {
			return val
		}
	}
	return ""
}

type proxyFromEnv struct {
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string

	http, https *url.URL
}

func (pfe *proxyFromEnv) initialize() {
	pfe.HTTPProxy = getEnvAny("HTTP_PROXY", "http_proxy")
	pfe.HTTPSProxy = getEnvAny("HTTPS_PROXY", "https_proxy")
	pfe.NoProxy = getEnvAny("NO_PROXY", "no_proxy")

	if pfe.NoProxy != "" {
		return
	}

	if pfe.HTTPProxy != "" {
		pfe.http, _ = url.Parse(pfe.HTTPProxy)
	}
	if pfe.HTTPSProxy != "" {
		pfe.https, _ = url.Parse(pfe.HTTPSProxy)
	}
}

func (pfe *proxyFromEnv) isLoopback(host string) (ok bool) {
	host, _, _ = net.SplitHostPort(host)
	if host == "localhost" {
		return true
	}
	return net.ParseIP(host).IsLoopback()
}

func (pfe *proxyFromEnv) Proxy(req *http.Request) (proxy *url.URL, err error) {
	if pfe.isLoopback(req.URL.Host) {
		switch req.URL.Scheme {
		case "http":
			return pfe.http, nil
		case "https":
			return pfe.https, nil
		default:
		}
	}
	return http.ProxyFromEnvironment(req)
}

var envProxy proxyFromEnv

var DefaultTransport = &http.Transport{
	Proxy: envProxy.Proxy,
	DialContext: (&net.Dialer{
		Timeout:   45 * time.Second,
		KeepAlive: 45 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	MaxIdleConnsPerHost:   100,
}

func init() {
	Client = &http.Client{
		Transport: DefaultTransport,
	}
	envProxy.initialize()
}

func NewHTTPRequest(method string, url string, data []byte) (*http.Request, context.Context, context.CancelFunc, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Access-Control-Allow-Origin", "*")
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*10)
	return req, ctx, cncl, err
}

func setClientInfo(req *http.Request) {
	req.Header.Set("X-App-Client-ID", client.GetClientID())
	req.Header.Set("X-App-Client-Key", client.GetClientPublicKey())
}

func setClientInfoWithSign(req *http.Request, allocation string) error {
	setClientInfo(req)

	sign, err := client.Sign(encryption.Hash(allocation))
	if err != nil {
		return err
	}
	req.Header.Set(CLIENT_SIGNATURE_HEADER, sign)

	return nil
}

func NewCommitRequest(baseUrl, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, COMMIT_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	setClientInfo(req)
	return req, nil
}

func NewReferencePathRequest(baseUrl, allocation string, paths []string) (*http.Request, error) {
	nurl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	nurl.Path += REFERENCE_ENDPOINT + allocation
	pathBytes, err := json.Marshal(paths)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Add("paths", string(pathBytes))
	//url := fmt.Sprintf("%s%s%s?path=%s", baseUrl, LIST_ENDPOINT, allocation, path)
	nurl.RawQuery = params.Encode() // Escape Query Parameters
	req, err := http.NewRequest(http.MethodGet, nurl.String(), nil)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewCalculateHashRequest(baseUrl, allocation string, paths []string) (*http.Request, error) {
	nurl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	nurl.Path += CALCULATE_HASH_ENDPOINT + allocation
	pathBytes, err := json.Marshal(paths)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Add("paths", string(pathBytes))
	nurl.RawQuery = params.Encode() // Escape Query Parameters
	req, err := http.NewRequest(http.MethodPost, nurl.String(), nil)
	if err != nil {
		return nil, err
	}
	setClientInfo(req)
	return req, nil
}

func NewObjectTreeRequest(baseUrl, allocation string, path string) (*http.Request, error) {
	nurl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	nurl.Path += OBJECT_TREE_ENDPOINT + allocation
	params := url.Values{}
	params.Add("path", path)
	//url := fmt.Sprintf("%s%s%s?path=%s", baseUrl, LIST_ENDPOINT, allocation, path)
	nurl.RawQuery = params.Encode() // Escape Query Parameters
	req, err := http.NewRequest(http.MethodGet, nurl.String(), nil)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewRefsRequest(baseUrl, allocationID, path, offsetPath, updatedDate, offsetDate, fileType, refType string, level, pageLimit int) (*http.Request, error) {
	nUrl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	nUrl.Path += REFS_ENDPOINT + allocationID
	params := url.Values{}
	params.Add("path", path)
	params.Add("offsetPath", offsetPath)
	params.Add("pageLimit", strconv.Itoa(pageLimit))
	params.Add("updatedDate", updatedDate)
	params.Add("offsetDate", offsetDate)
	params.Add("fileType", fileType)
	params.Add("refType", refType)
	params.Add("level", strconv.Itoa(level))
	nUrl.RawQuery = params.Encode()
	req, err := http.NewRequest(http.MethodGet, nUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocationID); err != nil {
		return nil, err
	}

	return req, nil
}

func NewAllocationRequest(baseUrl, allocation string) (*http.Request, error) {
	nurl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	nurl.Path += ALLOCATION_ENDPOINT
	params := url.Values{}
	params.Add("id", allocation)
	nurl.RawQuery = params.Encode() // Escape Query Parameters
	req, err := http.NewRequest(http.MethodGet, nurl.String(), nil)
	if err != nil {
		return nil, err
	}
	setClientInfo(req)
	return req, nil
}

func NewCommitMetaTxnRequest(baseUrl string, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, COMMIT_META_TXN_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	setClientInfo(req)
	return req, nil
}

func NewCollaboratorRequest(baseUrl string, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, COLLABORATOR_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func GetCollaboratorsRequest(baseUrl string, allocation string, query *url.Values) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s?%s", baseUrl, COLLABORATOR_ENDPOINT, allocation, query.Encode())
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func DeleteCollaboratorRequest(baseUrl string, allocation string, query *url.Values) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s?%s", baseUrl, COLLABORATOR_ENDPOINT, allocation, query.Encode())

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewFileMetaRequest(baseUrl string, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, FILE_META_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	err = setClientInfoWithSign(req, allocation)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func NewFileStatsRequest(baseUrl string, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, FILE_STATS_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewListRequest(baseUrl, allocation string, path, pathHash string, auth_token string) (*http.Request, error) {
	nurl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	nurl.Path += LIST_ENDPOINT + allocation
	params := url.Values{}
	params.Add("path", path)
	params.Add("path_hash", pathHash)
	params.Add("auth_token", auth_token)
	//url := fmt.Sprintf("%s%s%s?path=%s", baseUrl, LIST_ENDPOINT, allocation, path)
	nurl.RawQuery = params.Encode() // Escape Query Parameters
	req, err := http.NewRequest(http.MethodGet, nurl.String(), nil)
	if err != nil {
		return nil, err
	}
	setClientInfo(req)
	return req, nil
}

// NewUploadRequestWithMethod create a http reqeust of upload
func NewUploadRequestWithMethod(baseURL, allocation string, body io.Reader, method string) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseURL, UPLOAD_ENDPOINT, allocation)
	var req *http.Request
	var err error

	req, err = http.NewRequest(method, url, body)

	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewUploadRequest(baseUrl, allocation string, body io.Reader, update bool) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, UPLOAD_ENDPOINT, allocation)
	var req *http.Request
	var err error
	if update {
		req, err = http.NewRequest(http.MethodPut, url, body)
	} else {
		req, err = http.NewRequest(http.MethodPost, url, body)
	}
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewRenameRequest(baseUrl, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, RENAME_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewCopyRequest(baseUrl, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, COPY_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewDownloadRequest(baseUrl, allocation string) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, DOWNLOAD_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setClientInfo(req)
	return req, nil
}

func NewDeleteRequest(baseUrl, allocation string, query *url.Values) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s?%s", baseUrl, UPLOAD_ENDPOINT, allocation, query.Encode())

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewCreateDirRequest(baseUrl, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, DIR_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewShareRequest(baseUrl, allocation string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s", baseUrl, SHARE_ENDPOINT, allocation)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func NewRevokeShareRequest(baseUrl, allocation string, query *url.Values) (*http.Request, error) {
	url := fmt.Sprintf("%s%s%s?%s", baseUrl, SHARE_ENDPOINT, allocation, query.Encode())
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	if err := setClientInfoWithSign(req, allocation); err != nil {
		return nil, err
	}

	return req, nil
}

func MakeSCRestAPICall(scAddress string, relativePath string, params map[string]string, handler SCRestAPIHandler) ([]byte, error) {
	numSharders := len(blockchain.GetSharders())
	sharders := blockchain.GetSharders()
	responses := make(map[int]int)
	mu := &sync.Mutex{}
	entityResult := make(map[string][]byte)
	var retObj []byte
	maxCount := 0
	wg := sync.WaitGroup{}
	for _, sharder := range util.Shuffle(sharders) {
		wg.Add(1)
		go func(sharder string) {
			defer wg.Done()
			urlString := fmt.Sprintf("%v/%v%v%v", sharder, SC_REST_API_URL, scAddress, relativePath)
			urlObj, _ := url.Parse(urlString)
			q := urlObj.Query()
			for k, v := range params {
				q.Add(k, v)
			}
			urlObj.RawQuery = q.Encode()
			client := &http.Client{Transport: DefaultTransport}

			response, err := client.Get(urlObj.String())
			if err == nil {
				defer response.Body.Close()
				entityBytes, _ := ioutil.ReadAll(response.Body)

				mu.Lock()
				responses[response.StatusCode]++
				if responses[response.StatusCode] > maxCount {
					maxCount = responses[response.StatusCode]
					retObj = entityBytes
				}
				entityResult[sharder] = retObj
				mu.Unlock()
			}
		}(sharder)
	}
	wg.Wait()

	var err error
	rate := float32(maxCount*100) / float32(numSharders)
	if rate < consensusThresh {
		err = errors.New("consensus_failed", "consensus failed on sharders")
	}

	c := 0
	dominant := 200
	for code, count := range responses {
		if count > c {
			dominant = code
		}
	}

	if dominant != 200 {
		var objmap map[string]json.RawMessage
		err := json.Unmarshal(retObj, &objmap)
		if err != nil {
			return nil, errors.New("", string(retObj))
		}

		var parsed string
		err = json.Unmarshal(objmap["error"], &parsed)
		if err != nil || parsed == "" {
			return nil, errors.New("", string(retObj))
		}

		return nil, errors.New("", parsed)
	}

	if handler != nil {
		handler(entityResult, numSharders, err)
	}

	if rate > consensusThresh {
		return retObj, nil
	}
	return nil, err
}

func HttpDo(ctx context.Context, cncl context.CancelFunc, req *http.Request, f func(*http.Response, error) error) error {
	// Run the HTTP request in a goroutine and pass the response to f.
	c := make(chan error, 1)
	go func() { c <- f(Client.Do(req.WithContext(ctx))) }()
	// TODO: Check cncl context required in any case
	// defer cncl()
	select {
	case <-ctx.Done():
		DefaultTransport.CancelRequest(req)
		<-c // Wait for f to return.
		return ctx.Err()
	case err := <-c:
		return err
	}
}
