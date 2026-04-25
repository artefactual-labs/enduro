package publisher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type sftpPublisher struct {
	cfg      Config
	progress func(Progress)
}

func (p *sftpPublisher) Publish(ctx context.Context, localPath, relPath string) (*PublishedTransfer, error) {
	relPath, err := cleanRelPath(relPath)
	if err != nil {
		return nil, nonRetryable(err)
	}

	remotePath := path.Join(defaultString(p.cfg.RemoteDir, "/"), relPath)
	submittedPath := path.Join(p.cfg.SubmittedPathPrefix, relPath)

	if err := p.publish(ctx, localPath, remotePath); err != nil {
		return nil, err
	}

	return &PublishedTransfer{
		RelPath:    submittedPath,
		RemotePath: remotePath,
	}, nil
}

func (p *sftpPublisher) Delete(ctx context.Context, remotePath string) error {
	if remotePath == "" {
		return nil
	}

	conn, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := removeRemoteAll(conn.sftpClient, remotePath); err != nil {
		return fmt.Errorf("remove published remote transfer: %w", err)
	}

	return nil
}

func (p *sftpPublisher) publish(ctx context.Context, localPath, remotePath string) error {
	stat, err := os.Stat(localPath)
	if err != nil {
		return nonRetryable(LocalTransferMissingError{Path: localPath, err: err})
	}

	conn, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	remoteParent := path.Dir(remotePath)
	remoteBase := path.Base(remotePath)
	tempRemotePath := path.Join(remoteParent, "."+remoteBase+".uploading")

	if err := removeRemoteAll(conn.sftpClient, tempRemotePath); err != nil {
		return fmt.Errorf("remove temporary remote transfer: %w", err)
	}
	if err := conn.sftpClient.MkdirAll(remoteParent); err != nil {
		return fmt.Errorf("create remote transfer parent: %w", err)
	}

	if stat.IsDir() {
		err = p.uploadDir(ctx, conn.sftpClient, localPath, tempRemotePath)
	} else {
		err = p.uploadFile(ctx, conn.sftpClient, localPath, tempRemotePath, stat.Mode())
	}
	if err != nil {
		return err
	}

	if err := removeRemoteAll(conn.sftpClient, remotePath); err != nil {
		return fmt.Errorf("remove previous remote transfer: %w", err)
	}
	if err := conn.sftpClient.Rename(tempRemotePath, remotePath); err != nil {
		return fmt.Errorf("publish remote transfer: %w", err)
	}

	return nil
}

type sftpConnection struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func (c *sftpConnection) Close() error {
	sftpErr := c.sftpClient.Close()
	sshErr := c.sshClient.Close()
	if sftpErr != nil {
		return sftpErr
	}
	return sshErr
}

func (p *sftpPublisher) connect(ctx context.Context) (*sftpConnection, error) {
	port := p.cfg.Port
	if port == 0 {
		port = 22
	}

	callback, err := hostKeyCallback(p.cfg)
	if err != nil {
		return nil, nonRetryable(err)
	}

	auth, err := sftpAuthMethods(p.cfg)
	if err != nil {
		return nil, nonRetryable(err)
	}

	address := net.JoinHostPort(p.cfg.Host, strconv.Itoa(port))
	netConn, err := (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connect to SFTP server: %w", err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, address, &ssh.ClientConfig{
		User:            p.cfg.User,
		Auth:            auth,
		HostKeyCallback: callback,
		Timeout:         30 * time.Second,
	})
	if err != nil {
		_ = netConn.Close()
		if nonRetryableConnectError(err) {
			return nil, nonRetryable(fmt.Errorf("connect to SFTP server: %w", err))
		}
		return nil, fmt.Errorf("connect to SFTP server: %w", err)
	}
	sshClient := ssh.NewClient(sshConn, chans, reqs)

	client, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, fmt.Errorf("create SFTP client: %w", err)
	}

	return &sftpConnection{
		sshClient:  sshClient,
		sftpClient: client,
	}, nil
}

func sftpAuthMethods(cfg Config) ([]ssh.AuthMethod, error) {
	if cfg.PrivateKey.Path != "" {
		keyBytes, err := os.ReadFile(filepath.Clean(cfg.PrivateKey.Path)) // #nosec G304 -- path comes from administrator config.
		if err != nil {
			return nil, fmt.Errorf("read SFTP private key: %w", err)
		}

		var signer ssh.Signer
		if cfg.PrivateKey.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(cfg.PrivateKey.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("parse SFTP private key: %w", err)
		}

		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}

	return []ssh.AuthMethod{ssh.Password(cfg.Password)}, nil
}

func hostKeyCallback(cfg Config) (ssh.HostKeyCallback, error) {
	if cfg.InsecureIgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil // #nosec G106 -- allowed only when explicitly enabled by administrator config.
	}
	if cfg.KnownHostsFile != "" {
		return knownhosts.New(cfg.KnownHostsFile)
	}

	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(cfg.HostKey))
	if err != nil {
		return nil, fmt.Errorf("parse SFTP host key: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if !bytes.Equal(hostKey.Marshal(), key.Marshal()) {
			return fmt.Errorf("SFTP host key mismatch for %s", hostname)
		}
		return nil
	}, nil
}

func nonRetryableConnectError(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "ssh: unable to authenticate") ||
		strings.Contains(msg, "knownhosts: key is unknown") ||
		strings.Contains(msg, "knownhosts: key mismatch") ||
		strings.Contains(msg, "host key mismatch")
}

func (p *sftpPublisher) uploadDir(ctx context.Context, client *sftp.Client, localDir, remoteDir string) error {
	return filepath.WalkDir(localDir, func(localPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk local transfer: %w", err)
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return nonRetryable(fmt.Errorf("SFTP transfer publisher does not support symlinks: %s", localPath))
		}

		rel, err := filepath.Rel(localDir, localPath)
		if err != nil {
			return nonRetryable(fmt.Errorf("calculate local transfer path: %w", err))
		}

		remotePath := remoteDir
		if rel != "." {
			remotePath = path.Join(remoteDir, filepath.ToSlash(rel))
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat local transfer path: %w", err)
		}

		p.report(localPath, 0)

		if entry.IsDir() {
			if err := client.MkdirAll(remotePath); err != nil {
				return fmt.Errorf("create remote directory: %w", err)
			}
			_ = client.Chmod(remotePath, info.Mode())
			return nil
		}

		return p.uploadFile(ctx, client, localPath, remotePath, info.Mode())
	})
}

func (p *sftpPublisher) uploadFile(ctx context.Context, client *sftp.Client, localPath, remotePath string, mode os.FileMode) error {
	p.report(localPath, 0)

	if err := client.MkdirAll(path.Dir(remotePath)); err != nil {
		return fmt.Errorf("create remote file parent: %w", err)
	}

	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local transfer file: %w", err)
	}
	defer src.Close()

	dst, err := client.OpenFile(remotePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("create remote transfer file: %w", err)
	}

	_, copyErr := io.Copy(dst, newProgressReader(ctx, src, localPath, p.report))
	closeErr := dst.Close()
	if copyErr != nil {
		return fmt.Errorf("upload transfer file: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close remote transfer file: %w", closeErr)
	}

	_ = client.Chmod(remotePath, mode)

	return nil
}

func (p *sftpPublisher) report(localPath string, bytes int64) {
	if p.progress != nil {
		p.progress(Progress{LocalPath: localPath, Bytes: bytes})
	}
}

func removeRemoteAll(client *sftp.Client, remotePath string) error {
	if err := client.RemoveAll(remotePath); err != nil {
		if errorsIsNotExist(err) {
			return nil
		}
		return err
	}

	return nil
}

func errorsIsNotExist(err error) bool {
	return os.IsNotExist(err) || strings.Contains(err.Error(), "does not exist")
}

type progressReader struct {
	ctx       context.Context
	reader    io.Reader
	localPath string
	report    func(string, int64)
	copied    int64
	lastSent  time.Time
}

func newProgressReader(ctx context.Context, reader io.Reader, localPath string, report func(string, int64)) *progressReader {
	return &progressReader{
		ctx:       ctx,
		reader:    reader,
		localPath: localPath,
		report:    report,
		lastSent:  time.Now(),
	}
}

func (r *progressReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	n, err := r.reader.Read(p)
	if n > 0 {
		r.copied += int64(n)
		now := time.Now()
		if now.Sub(r.lastSent) >= 5*time.Second {
			r.report(r.localPath, r.copied)
			r.lastSent = now
		}
	}

	return n, err
}
