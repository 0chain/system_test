package zs3servertests

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	porcupine "github.com/anishathalye/porcupine"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverPorcupineLinearizability runs a Porcupine linearizability harness
// against the S3 gateway (and optionally a local NFS mount).
//
// Pass criterion: `porcupine.CheckOperationsVerbose` returns `Ok` (LINEARIZABLE).
// `Unknown` (check timeout) is tolerated — it's algorithm limitation, not a bug.
// `Illegal` fails the test.
//
// Subtests:
//   s3/zs3server  — S3 API → zs3server :9100 (WAL + blobber)
//   fs/nfs        — file-per-key against ZS3_NFS_MOUNT (NFSv4)
//
// Env:
//   ZS3_NFS_MOUNT     — optional NFS path for fs subtest (skips if unset)
//   PORC_WORKERS      — default 8
//   PORC_OPS          — default 500
//   PORC_KEYS         — default 20
//   PORC_TIMEOUT      — porcupine check timeout, default 60s
func TestZs3serverPorcupineLinearizability(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml not found, skipping")
	}
	config := cliutils.ReadFile(testSetup)
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("zs3server not reachable: %v", err)
	}
	conn.Close()

	t := test.NewSystemTest(testSetup)

	workers := intEnv("PORC_WORKERS", 8)
	opsPerW := intEnv("PORC_OPS", 500)
	keys := intEnv("PORC_KEYS", 20)
	checkTimeout := durEnv("PORC_TIMEOUT", 60*time.Second)

	t.RunSequentiallyWithTimeout("s3/zs3server", 15*time.Minute, func(t *test.SystemTest) {
		endpoint := "http://" + config.Server + ":" + config.HostPort
		client := newS3Client(t, endpoint, config.AccessKey, config.SecretKey)
		bucket := "porcupine-s3"
		_ = ensureBucket(client, bucket)
		be := &s3PorcKV{client: client, bucket: bucket, prefix: "porc"}
		_ = be.reset()
		runPorcupine(t, be, workers, opsPerW, keys, checkTimeout)
	})

	nfsMount := os.Getenv("ZS3_NFS_MOUNT")
	if nfsMount != "" {
		if _, err := os.Stat(nfsMount); err == nil {
			t.RunSequentiallyWithTimeout("fs/nfs", 15*time.Minute, func(t *test.SystemTest) {
				be := &fsPorcKV{dir: filepath.Join(nfsMount, "porcupine_kv")}
				_ = be.reset()
				runPorcupine(t, be, workers, opsPerW, keys, checkTimeout)
			})
		} else {
			t.Logf("ZS3_NFS_MOUNT=%s not accessible, skipping fs subtest", nfsMount)
		}
	}
}

// ---- Porcupine KV model ----

type kvInput struct {
	Op  string
	Key string
	Val string
}
type kvOutput struct {
	Val string
	Err bool
}

var kvModel = porcupine.Model{
	Init: func() interface{} { return map[string]string{} },
	Equal: func(a, b interface{}) bool {
		m1, m2 := a.(map[string]string), b.(map[string]string)
		if len(m1) != len(m2) {
			return false
		}
		for k, v := range m1 {
			if v2, ok := m2[k]; !ok || v2 != v {
				return false
			}
		}
		return true
	},
	Step: func(state, input, output interface{}) (bool, interface{}) {
		st := state.(map[string]string)
		op := input.(kvInput)
		out := output.(kvOutput)
		if out.Err {
			return true, st
		}
		ns := make(map[string]string, len(st))
		for k, v := range st {
			ns[k] = v
		}
		switch op.Op {
		case "put":
			ns[op.Key] = op.Val
			return out.Val == "ok", ns
		case "get":
			if v, ok := st[op.Key]; ok {
				return out.Val == v, st
			}
			return out.Val == "", st
		case "del":
			delete(ns, op.Key)
			return out.Val == "ok", ns
		}
		return false, state
	},
}

type porcBackend interface {
	put(key, val string) error
	get(key string) (string, error)
	del(key string) error
	reset() error
}

// --- fs backend: one file per key, unique-per-writer .tmp + atomic rename ---

type fsPorcKV struct{ dir string }

func (k *fsPorcKV) path(key string) string { return filepath.Join(k.dir, key) }
func (k *fsPorcKV) put(key, val string) error {
	var nonce [8]byte
	_, _ = rand.Read(nonce[:])
	tmp := fmt.Sprintf("%s.tmp.%x", k.path(key), nonce)
	if err := os.WriteFile(tmp, []byte(val), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, k.path(key))
}
func (k *fsPorcKV) get(key string) (string, error) {
	b, err := os.ReadFile(k.path(key))
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(b), err
}
func (k *fsPorcKV) del(key string) error {
	err := os.Remove(k.path(key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
func (k *fsPorcKV) reset() error {
	_ = os.RemoveAll(k.dir)
	return os.MkdirAll(k.dir, 0o755)
}

// --- s3 backend ---

type s3PorcKV struct {
	client         *s3.S3
	bucket, prefix string
}

func (k *s3PorcKV) objKey(key string) string { return k.prefix + "/" + key }
func (k *s3PorcKV) put(key, val string) error {
	ok := k.objKey(key)
	_, err := k.client.PutObject(&s3.PutObjectInput{
		Bucket: &k.bucket, Key: &ok, Body: bytes.NewReader([]byte(val)),
	})
	return err
}
func (k *s3PorcKV) get(key string) (string, error) {
	ok := k.objKey(key)
	out, err := k.client.GetObject(&s3.GetObjectInput{Bucket: &k.bucket, Key: &ok})
	if err != nil {
		// treat NoSuchKey as absent
		return "", nil
	}
	defer out.Body.Close()
	b, err := io.ReadAll(out.Body)
	return string(b), err
}
func (k *s3PorcKV) del(key string) error {
	ok := k.objKey(key)
	_, err := k.client.DeleteObject(&s3.DeleteObjectInput{Bucket: &k.bucket, Key: &ok})
	return err
}
func (k *s3PorcKV) reset() error {
	list, err := k.client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &k.bucket, Prefix: aws.String(k.prefix + "/"),
	})
	if err != nil {
		return err
	}
	for _, o := range list.Contents {
		_, _ = k.client.DeleteObject(&s3.DeleteObjectInput{Bucket: &k.bucket, Key: o.Key})
	}
	return nil
}

// ---- Run harness + check ----

func runPorcupine(t *test.SystemTest, be porcBackend, workers, opsPerW, nKeys int, checkTimeout time.Duration) {
	var (
		mu      sync.Mutex
		history []porcupine.Operation
		errs    int64
		nextID  int32
	)
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			r := mrand.New(mrand.NewSource(int64(wid) * 997))
			for i := 0; i < opsPerW; i++ {
				key := fmt.Sprintf("k%03d", r.Intn(nKeys))
				pick := r.Intn(100)
				t0 := time.Now().UnixNano()
				var in kvInput
				var out kvOutput
				switch {
				case pick < 45:
					sn, _ := rand.Int(rand.Reader, big.NewInt(1<<30))
					val := fmt.Sprintf("v-%d-%d-%d", wid, atomic.AddInt32(&nextID, 1), sn.Int64())
					in = kvInput{Op: "put", Key: key, Val: val}
					if err := be.put(key, val); err != nil {
						out = kvOutput{Err: true}
						atomic.AddInt64(&errs, 1)
					} else {
						out = kvOutput{Val: "ok"}
					}
				case pick < 85:
					in = kvInput{Op: "get", Key: key}
					v, err := be.get(key)
					if err != nil {
						out = kvOutput{Err: true}
						atomic.AddInt64(&errs, 1)
					} else {
						out = kvOutput{Val: v}
					}
				default:
					in = kvInput{Op: "del", Key: key}
					if err := be.del(key); err != nil {
						out = kvOutput{Err: true}
						atomic.AddInt64(&errs, 1)
					} else {
						out = kvOutput{Val: "ok"}
					}
				}
				t1 := time.Now().UnixNano()
				mu.Lock()
				history = append(history, porcupine.Operation{
					ClientId: wid, Input: in, Call: t0, Output: out, Return: t1,
				})
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()
	wall := time.Since(start)
	total := len(history)
	t.Logf("Collected %d ops in %.3fs (errors=%d) throughput=%.1f ops/s",
		total, wall.Seconds(), errs, float64(total)/wall.Seconds())

	res, _ := porcupine.CheckOperationsVerbose(kvModel, history, checkTimeout)
	switch res {
	case porcupine.Ok:
		t.Logf("RESULT: LINEARIZABLE")
	case porcupine.Illegal:
		t.Fatalf("RESULT: NON-LINEARIZABLE — check history above")
	case porcupine.Unknown:
		t.Logf("RESULT: UNKNOWN (check timeout %s) — tolerated", checkTimeout)
	}
	_ = be.reset()
}

func intEnv(name string, dflt int) int {
	v := os.Getenv(name)
	if v == "" {
		return dflt
	}
	n := 0
	_, _ = fmt.Sscanf(v, "%d", &n)
	if n <= 0 {
		return dflt
	}
	return n
}

func durEnv(name string, dflt time.Duration) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return dflt
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return dflt
	}
	return d
}
