package publisher

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"gotest.tools/v3/assert"
)

func TestSFTPPublisherPublishAndDeleteWithPassword(t *testing.T) {
	t.Parallel()

	server := startSFTPServer(t, testSFTPServerConfig{
		password: "12345",
	})

	localDir := t.TempDir()
	assert.NilError(t, os.MkdirAll(filepath.Join(localDir, "objects"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(localDir, "objects", "hello.txt"), []byte("hello"), 0o600))

	var progress []Progress
	pub, err := New(Config{
		Type:                  "sftp",
		Host:                  server.host,
		Port:                  server.port,
		User:                  "archivematica",
		Password:              "12345",
		RemoteDir:             "incoming",
		SubmittedPathPrefix:   "archivematica/transfers",
		InsecureIgnoreHostKey: true,
	}, WithProgress(func(p Progress) {
		progress = append(progress, p)
	}))
	assert.NilError(t, err)

	res, err := pub.Publish(context.Background(), localDir, "transfer")
	assert.NilError(t, err)
	assert.DeepEqual(t, res, &PublishedTransfer{
		RelPath:    "archivematica/transfers/transfer",
		RemotePath: "incoming/transfer",
	})
	assert.Assert(t, len(progress) > 0)
	assert.Equal(t, readFile(t, filepath.Join(server.root, "incoming", "transfer", "objects", "hello.txt")), "hello")
	assert.Assert(t, !pathExists(filepath.Join(server.root, "incoming", ".transfer.uploading")))

	assert.NilError(t, pub.Delete(context.Background(), res.RemotePath))
	assert.Assert(t, !pathExists(filepath.Join(server.root, "incoming", "transfer")))
}

func TestSFTPPublisherPublishWithPrivateKeyAndKnownHosts(t *testing.T) {
	t.Parallel()

	clientSigner, privateKeyPath := writePrivateKey(t)
	server := startSFTPServer(t, testSFTPServerConfig{
		authorizedKey: clientSigner.PublicKey(),
	})
	knownHostsFile := writeKnownHosts(t, server)

	localDir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(localDir, "transfer.zip"), []byte("zip"), 0o600))

	pub, err := New(Config{
		Type:                "sftp",
		Host:                server.host,
		Port:                server.port,
		User:                "archivematica",
		RemoteDir:           ".",
		SubmittedPathPrefix: "archivematica/transfers",
		KnownHostsFile:      knownHostsFile,
		PrivateKey: PrivateKeyConfig{
			Path: privateKeyPath,
		},
	})
	assert.NilError(t, err)

	res, err := pub.Publish(context.Background(), filepath.Join(localDir, "transfer.zip"), "transfer.zip")
	assert.NilError(t, err)
	assert.DeepEqual(t, res, &PublishedTransfer{
		RelPath:    "archivematica/transfers/transfer.zip",
		RemotePath: "transfer.zip",
	})
	assert.Equal(t, readFile(t, filepath.Join(server.root, "transfer.zip")), "zip")
}

func TestSFTPPublisherRejectsUnauthorizedKey(t *testing.T) {
	t.Parallel()

	_, privateKeyPath := writePrivateKey(t)
	server := startSFTPServer(t, testSFTPServerConfig{
		authorizedKey: mustSigner(t).PublicKey(),
	})
	knownHostsFile := writeKnownHosts(t, server)

	pub, err := New(Config{
		Type:           "sftp",
		Host:           server.host,
		Port:           server.port,
		User:           "archivematica",
		KnownHostsFile: knownHostsFile,
		PrivateKey: PrivateKeyConfig{
			Path: privateKeyPath,
		},
	})
	assert.NilError(t, err)

	_, err = pub.Publish(context.Background(), t.TempDir(), "transfer")
	assert.ErrorContains(t, err, "connect to SFTP server")
	assert.Assert(t, IsNonRetryable(err))
}

func TestSFTPPublisherRejectsHostKeyMismatch(t *testing.T) {
	t.Parallel()

	server := startSFTPServer(t, testSFTPServerConfig{
		password: "12345",
	})
	knownHostsFile := writeKnownHosts(t, server, mustSigner(t).PublicKey())

	pub, err := New(Config{
		Type:           "sftp",
		Host:           server.host,
		Port:           server.port,
		User:           "archivematica",
		Password:       "12345",
		KnownHostsFile: knownHostsFile,
	})
	assert.NilError(t, err)

	_, err = pub.Publish(context.Background(), t.TempDir(), "transfer")
	assert.ErrorContains(t, err, "connect to SFTP server")
	assert.Assert(t, IsNonRetryable(err))
}

func TestSFTPPublisherReportsMissingLocalTransfer(t *testing.T) {
	t.Parallel()

	pub, err := New(Config{
		Type:                  "sftp",
		Host:                  "127.0.0.1",
		Port:                  22,
		User:                  "archivematica",
		Password:              "12345",
		InsecureIgnoreHostKey: true,
	})
	assert.NilError(t, err)

	_, err = pub.Publish(context.Background(), filepath.Join(t.TempDir(), "missing"), "transfer")
	assert.Assert(t, IsNonRetryable(err))
	assert.Assert(t, IsLocalTransferMissing(err))
}

type testSFTPServerConfig struct {
	password      string
	authorizedKey ssh.PublicKey
}

type testSFTPServer struct {
	host      string
	port      int
	root      string
	hostKey   ssh.PublicKey
	closeOnce sync.Once
	close     func() error
}

func startSFTPServer(t *testing.T, cfg testSFTPServerConfig) *testSFTPServer {
	t.Helper()

	root := t.TempDir()
	hostSigner := mustSigner(t)

	sshConfig := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if cfg.password != "" && string(password) == cfg.password {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected")
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if cfg.authorizedKey != nil && string(key.Marshal()) == string(cfg.authorizedKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("public key rejected")
		},
	}
	sshConfig.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)

	server := &testSFTPServer{
		host:    "127.0.0.1",
		port:    listener.Addr().(*net.TCPAddr).Port,
		root:    root,
		hostKey: hostSigner.PublicKey(),
		close:   listener.Close,
	}
	t.Cleanup(func() {
		server.closeOnce.Do(func() {
			_ = server.close()
		})
	})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go serveSFTPConn(conn, sshConfig, root)
		}
	}()

	return server
}

func serveSFTPConn(conn net.Conn, cfg *ssh.ServerConfig, root string) {
	_, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return
	}
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go func() {
			defer channel.Close()
			for req := range requests {
				if req.Type != "subsystem" || parseSubsystem(req.Payload) != "sftp" {
					_ = req.Reply(false, nil)
					continue
				}
				_ = req.Reply(true, nil)

				server, err := sftp.NewServer(channel, sftp.WithServerWorkingDirectory(root))
				if err != nil {
					return
				}
				_ = server.Serve()
				return
			}
		}()
	}
}

func parseSubsystem(payload []byte) string {
	if len(payload) < 4 {
		return ""
	}
	size := binary.BigEndian.Uint32(payload[:4])
	if int(size) > len(payload)-4 {
		return ""
	}
	return string(payload[4 : 4+size])
}

func writePrivateKey(t *testing.T) (ssh.Signer, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)
	signer, err := ssh.NewSignerFromKey(privateKey)
	assert.NilError(t, err)

	privateKeyPath := filepath.Join(t.TempDir(), "id_rsa")
	assert.NilError(t, os.WriteFile(privateKeyPath, pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}), 0o600))

	return signer, privateKeyPath
}

func mustSigner(t *testing.T) ssh.Signer {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)
	signer, err := ssh.NewSignerFromKey(privateKey)
	assert.NilError(t, err)

	return signer
}

func writeKnownHosts(t *testing.T, server *testSFTPServer, keys ...ssh.PublicKey) string {
	t.Helper()

	hostKey := server.hostKey
	if len(keys) > 0 {
		hostKey = keys[0]
	}

	path := filepath.Join(t.TempDir(), "known_hosts")
	line := knownhosts.Line([]string{net.JoinHostPort(server.host, strconv.Itoa(server.port))}, hostKey)
	assert.NilError(t, os.WriteFile(path, []byte(line+"\n"), 0o600))

	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	assert.NilError(t, err)

	return string(data)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
